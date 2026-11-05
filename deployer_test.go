package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
	"github.com/treejadey/lockdeader/vpk"
)

func TestDeployer(t *testing.T) {
	db, err := appdb.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	ctx := context.Background()
	mod := appdb.Mod{Section: "Mod", GameBananaID: 1, Name: "A", Enabled: true, LoadOrder: 2, InstalledAt: time.Now(), Files: []appdb.ModFile{{
		GameBananaID: 10, Name: "a.zip", MD5: "00", UploadedAt: time.Now(), DownloadedAt: time.Now(),
		VPKs: []appdb.VPK{{Path: "a.vpk", SHA256: "aa", Size: 1, Enabled: true}, {Path: "b.vpk", SHA256: "bb", Size: 2, Enabled: false}},
	}}}
	off := mod
	off.GameBananaID, off.Enabled, off.LoadOrder = 2, false, 1
	off.Files = []appdb.ModFile{{GameBananaID: 11, Name: "c.zip", MD5: "00", VPKs: []appdb.VPK{{Path: "c.vpk", SHA256: "cc", Size: 3, Enabled: true}}}}
	first := mod
	first.GameBananaID, first.LoadOrder = 3, 0
	first.Files = []appdb.ModFile{{GameBananaID: 12, Name: "d.zip", MD5: "00", VPKs: []appdb.VPK{{Path: "d.vpk", SHA256: "dd", Size: 4, Enabled: true}}}}

	for _, m := range []appdb.Mod{mod, off, first} {
		if _, err := db.AddMod(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	// Everything below is touched on the main thread only.
	store := install.Store{Dir: "/store"}
	gameDir := "/games/Deadlock"
	var applied [][]deploy.File
	var enabled []bool
	disabled := 0
	running := false
	applyErr := error(nil)
	var done []error
	var d *deployer

	mainthread.Wait(func() {
		d = newDeployer(db, store, func() string { return gameDir }, func() bool { return true }, "/data/gameinfo.gi.bak")
		d.apply = func(dir string, files []deploy.File) (deploy.Result, error) {
			if dir != gameDir {
				t.Errorf("applied to %q", dir)
			}

			applied = append(applied, files)

			return deploy.Result{}, applyErr
		}
		d.running = func() (bool, error) { return running, nil }
		d.enable = func(dir string, addons bool, backup string) (bool, error) {
			if dir != gameDir || backup != "/data/gameinfo.gi.bak" {
				t.Errorf("enabled in %q with backup %q", dir, backup)
			}

			enabled = append(enabled, addons)

			return true, nil
		}
		d.disable = func(dir, backup string) (bool, error) {
			disabled++

			return true, nil
		}
		d.onDone = func(err error) { done = append(done, err) }
	})

	doneCount := func(n int) func() bool { return func() bool { return len(done) == n } }

	// The files are the enabled VPKs of the enabled mods, in load order.
	var got error
	mainthread.Wait(func() { d.Apply(func(err error) { got = err }) })
	waitFor(t, "the first apply", doneCount(1))

	want := fmt.Sprint([]deploy.File{{SHA256: "dd", Path: store.Path("dd"), Size: 4}, {SHA256: "aa", Path: store.Path("aa"), Size: 1}})

	mainthread.Wait(func() {
		if len(applied) != 1 || fmt.Sprint(applied[0]) != want || got != nil || done[0] != nil || !d.Idle() {
			t.Errorf("applied %v (err %v, done %v), want %s", applied, got, done, want)
		}

		// A change applies half a second later, once.
		d.ApplySoon()
		d.ApplySoon()

		if d.Idle() {
			t.Error("idle with an apply waiting")
		}
	})

	waitFor(t, "the debounced apply", doneCount(2))

	// While the game runs, nothing is applied, and that's reported as such.
	mainthread.Wait(func() {
		running = true
		d.Apply(func(err error) { got = err })
	})
	waitFor(t, "the apply with the game running", doneCount(3))

	mainthread.Wait(func() {
		if len(applied) != 2 || !errors.Is(got, deploy.ErrGameRunning) {
			t.Errorf("applied %d times with the game running, err %v", len(applied), got)
		}

		// A failure is wrapped as the game folder's.
		running = false
		applyErr = deploy.ErrMissing
		d.Apply(func(err error) { got = err })
	})
	waitFor(t, "the failed apply", doneCount(4))

	// Preparing with mods applies, then puts the lines in; without, only
	// takes them out.
	var prepared []error

	mainthread.Wait(func() {
		if !errors.Is(got, errGameFolder) || !errors.Is(got, deploy.ErrMissing) {
			t.Errorf("err = %v", got)
		}

		applyErr = nil
		d.Prepare(true, func(err error) { prepared = append(prepared, err) })
	})
	waitFor(t, "preparing with mods", func() bool { return len(prepared) == 1 })

	mainthread.Wait(func() {
		if len(applied) != 4 || fmt.Sprint(enabled) != "[true]" || prepared[0] != nil {
			t.Errorf("after preparing with mods: applied %d, enabled %v, err %v", len(applied), enabled, prepared[0])
		}

		d.Prepare(false, func(err error) { prepared = append(prepared, err) })
	})
	waitFor(t, "preparing without mods", func() bool { return len(prepared) == 2 })

	mainthread.Wait(func() {
		if len(applied) != 4 || disabled != 1 || prepared[1] != nil {
			t.Errorf("after preparing without mods: applied %d, disabled %d, err %v", len(applied), disabled, prepared[1])
		}

		// Resetting takes the lines out, and applies, as part of the same
		// apply, so a launch that follows straight away gets them back.
		d.ResetSoon()
		d.Prepare(true, func(err error) { prepared = append(prepared, err) })
	})
	waitFor(t, "preparing after a reset", func() bool { return len(prepared) == 3 })

	mainthread.Wait(func() {
		if len(applied) != 5 || disabled != 2 || fmt.Sprint(enabled) != "[true true]" || prepared[2] != nil {
			t.Errorf("after a reset: applied %d, disabled %d, enabled %v, err %v", len(applied), disabled, enabled, prepared[2])
		}

		// Only the next apply takes them out.
		d.Apply(func(err error) { got = err })
	})
	waitFor(t, "the apply after a reset", doneCount(7))

	mainthread.Wait(func() {
		if len(applied) != 6 || disabled != 2 {
			t.Errorf("applied %d, disabled %d; want the lines left alone", len(applied), disabled)
		}

		// Without a game folder there's nothing to do.
		gameDir = ""
		got = errors.New("unset")
		d.Apply(func(err error) { got = err })

		if got != nil || len(applied) != 6 {
			t.Errorf("without a folder: err %v, applied %d", got, len(applied))
		}
	})
}

// Mods that share a sound event table get it merged, in a VPK of its own
// that loads first.
func TestDeployerMergesTables(t *testing.T) {
	db, err := appdb.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	store := install.Store{Dir: t.TempDir()}
	game := t.TempDir()
	heroes := map[string]string{"Haze": "sounds/vo/haze/select.vsnd", "Lash": "sounds/vo/lash/select.vsnd"}

	addVPK := func(files map[string][]byte) (string, int64) {
		var b bytes.Buffer
		if err := vpk.Write(&b, vpkSources(files)); err != nil {
			t.Fatal(err)
		}

		sha, size, err := store.Add(&b)
		if err != nil {
			t.Fatal(err)
		}

		return sha, size
	}

	// The game's pak01, written as Write writes any VPK.
	if err := os.MkdirAll(filepath.Dir(deploy.BasePak(game)), 0o755); err != nil {
		t.Fatal(err)
	}

	gameSHA, _ := addVPK(map[string][]byte{pickTable: soundTable(t, heroes)})
	if err := os.Rename(store.Path(gameSHA), deploy.BasePak(game)); err != nil {
		t.Fatal(err)
	}

	var shas []string

	for i, hero := range []string{"Haze", "Lash"} {
		table := maps.Clone(heroes)
		table[hero] = "sounds/vo/new/" + hero + ".vsnd"

		sha, size := addVPK(map[string][]byte{pickTable: soundTable(t, table), table[hero] + "_c": []byte(hero)})
		shas = append(shas, sha)

		m := appdb.Mod{Section: "Mod", GameBananaID: i + 1, Name: hero, Enabled: true, LoadOrder: i, InstalledAt: time.Now(),
			Files: []appdb.ModFile{{GameBananaID: 10 + i, Name: hero + ".zip", MD5: "00",
				VPKs: []appdb.VPK{{Path: "pak01_dir.vpk", SHA256: sha, Size: size, Enabled: true}}}}}

		if _, err := db.AddMod(context.Background(), m); err != nil {
			t.Fatal(err)
		}
	}

	var applied [][]deploy.File
	var done []error
	var d *deployer

	mainthread.Wait(func() {
		d = newDeployer(db, store, func() string { return game }, func() bool { return true }, "")
		d.apply = func(dir string, files []deploy.File) (deploy.Result, error) {
			applied = append(applied, files)

			return deploy.Result{}, nil
		}
		d.running = func() (bool, error) { return false, nil }
		d.onDone = func(err error) { done = append(done, err) }
		d.Apply(nil)
	})
	waitFor(t, "the apply", func() bool { return len(done) == 1 })

	mainthread.Wait(func() { d.Apply(nil) })
	waitFor(t, "the second apply", func() bool { return len(done) == 2 })

	mainthread.Wait(func() {
		files := applied[0]

		if len(files) != 3 || files[1].SHA256 != shas[0] || files[2].SHA256 != shas[1] {
			t.Fatalf("applied %v, want the merged tables, then the two mods", files)
		}

		// The same mods make the same file.
		if applied[1][0] != files[0] {
			t.Errorf("second apply's merged tables = %v, want %v again", applied[1][0], files[0])
		}

		data, err := os.ReadFile(files[0].Path)
		if err != nil {
			t.Fatal(err)
		}

		d, err := vpk.ReadDir(bytes.NewReader(data))
		if err != nil || len(d.Entries) != 1 || d.Entries[0].Path != pickTable {
			t.Fatalf("merged VPK: %v, %+v; want just the pick table", err, d)
		}

		table, _ := io.ReadAll(d.Open(d.Entries[0]))

		for _, want := range []string{"sounds/vo/new/Haze.vsnd", "sounds/vo/new/Lash.vsnd"} {
			if !bytes.Contains(table, []byte(want)) {
				t.Errorf("merged table lacks %s", want)
			}
		}
	})
}
