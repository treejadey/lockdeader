package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
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

		// Without a game folder there's nothing to do.
		gameDir = ""
		got = errors.New("unset")
		d.Apply(func(err error) { got = err })

		if got != nil || len(applied) != 4 {
			t.Errorf("without a folder: err %v, applied %d", got, len(applied))
		}
	})
}
