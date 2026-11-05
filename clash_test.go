package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"hash/crc32"
	"io"
	"maps"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/vpk"
)

// clashMod is a mod with one VPK holding paths, already in the store, as
// AddInstalled and the database take it.
func (it *installTest) clashMod(t *testing.T, id int, name string, enabled bool, paths ...string) appdb.Mod {
	t.Helper()

	var sources []vpk.Source

	for _, p := range paths {
		data := []byte(p)
		sources = append(sources, vpk.Source{
			Path: p, CRC32: crc32.ChecksumIEEE(data), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
		})
	}

	var b bytes.Buffer
	if err := vpk.Write(&b, sources); err != nil {
		t.Fatal(err)
	}

	sha, size, err := it.store.Add(&b)
	if err != nil {
		t.Fatal(err)
	}

	return appdb.Mod{
		Section: string(banana.SectionMods), GameBananaID: id, Name: name, Enabled: enabled,
		InstalledAt: time.Unix(1_790_000_000, 0),
		Files: []appdb.ModFile{{
			GameBananaID: id * 10, Name: name + ".zip",
			VPKs: []appdb.VPK{{Path: "pak01_dir.vpk", SHA256: sha, Size: size, Enabled: true}},
		}},
	}
}

func (it *installTest) enabled(t *testing.T) map[string]bool {
	t.Helper()

	on := make(map[string]bool)

	for _, m := range it.mods(t) {
		on[m.Name] = m.Enabled
	}

	return on
}

func (it *installTest) clashesOf(id int) string {
	return it.role(installedKey(banana.SectionMods, id), roleInstalledClashes).ToString()
}

func addInstalled(t *testing.T, it *installTest, mods ...appdb.Mod) {
	t.Helper()

	for _, m := range mods {
		if err := mainthread.Wait2(func() error { return it.AddInstalled(m) }); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTurningOnTurnsOffClashingMods(t *testing.T) {
	it := newInstallTest(t)

	gold := it.clashMod(t, 1, "Abrams Gold", true, "models/abrams.vmdl_c", "materials/gold.vmat_c")
	chrome := it.clashMod(t, 2, "Abrams Chrome", false, "Models/Abrams.vmdl_c", "materials/chrome.vmat_c")
	click := it.clashMod(t, 3, "Click Sound", true, "sounds/click.vsnd_c")
	addInstalled(t, it, gold, chrome, click)

	if got := it.clashesOf(1); got != "Abrams Chrome" {
		t.Errorf("clashes of Abrams Gold = %q", got)
	}

	if got := it.clashesOf(2); got != "Abrams Gold" {
		t.Errorf("clashes of Abrams Chrome = %q", got)
	}

	if got := it.clashesOf(3); got != "" {
		t.Errorf("clashes of Click Sound = %q", got)
	}

	// Turning one on turns the one it clashes with off, and nothing else.
	mainthread.Wait(func() { it.SetEnabled(installedKey(banana.SectionMods, 2), true) })

	want := map[string]bool{"Abrams Gold": false, "Abrams Chrome": true, "Click Sound": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after turning on Abrams Chrome: %v, want %v", got, want)
	}

	if it.role(installedKey(banana.SectionMods, 1), roleInstalledEnabled).ToBool() {
		t.Error("Abrams Gold's row still shows it on")
	}

	if got := it.message(); got != "Turned off Abrams Gold, which changes the same files" {
		t.Errorf("message = %q", got)
	}

	// Undo puts both back.
	mainthread.Wait(func() { it.Undo(it.count("messageId")) })

	want = map[string]bool{"Abrams Gold": true, "Abrams Chrome": false, "Click Sound": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after Undo: %v, want %v", got, want)
	}

	// Turning one off, or on when nothing it clashes with is on, says nothing.
	before := it.count("messageId")

	mainthread.Wait(func() {
		it.SetEnabled(installedKey(banana.SectionMods, 1), false)
		it.SetEnabled(installedKey(banana.SectionMods, 2), true)
	})

	if it.count("messageId") != before {
		t.Errorf("message = %q, want none", it.message())
	}
}

func TestTurningOnTurnsOffSeveral(t *testing.T) {
	it := newInstallTest(t)

	addInstalled(t, it,
		it.clashMod(t, 1, "Red", true, "a.vmat_c"),
		it.clashMod(t, 2, "Blue", true, "b.vmat_c"),
		it.clashMod(t, 3, "Both", false, "a.vmat_c", "b.vmat_c"))

	mainthread.Wait(func() { it.SetEnabled(installedKey(banana.SectionMods, 3), true) })

	if got := it.message(); got != "Turned off 2 mods that change the same files" {
		t.Errorf("message = %q", got)
	}

	if got := it.clashesOf(3); got != "Red, Blue" {
		t.Errorf("clashes of Both = %q", got)
	}

	// Once the message has gone, so has its Undo.
	id := it.count("messageId")
	mainthread.Wait(func() {
		it.messageGone(id)
		it.Undo(id)
	})

	want := map[string]bool{"Red": false, "Blue": false, "Both": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after Undo too late: %v, want %v", got, want)
	}
}

// A mod added turned on, as a take-over or an install is, wins over those
// it clashes with.
func TestAddedModTurnsOffClashing(t *testing.T) {
	it := newInstallTest(t)

	addInstalled(t, it,
		it.clashMod(t, 1, "Old", true, "a.vmat_c"),
		it.clashMod(t, 2, "New", true, "a.vmat_c"))

	want := map[string]bool{"Old": false, "New": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after adding New: %v, want %v", got, want)
	}
}

func TestNoiseDoesntClash(t *testing.T) {
	it := newInstallTest(t)

	addInstalled(t, it,
		it.clashMod(t, 1, "One", true, "addoninfo.txt", "readme.txt", "one.vmat_c"),
		it.clashMod(t, 2, "Two", true, "addoninfo.txt", "readme.txt", "two.vmat_c"))

	want := map[string]bool{"One": true, "Two": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("mods sharing only noise: %v, want %v", got, want)
	}

	if it.clashesOf(1) != "" || it.clashesOf(2) != "" {
		t.Errorf("clashes = %q and %q, want none", it.clashesOf(1), it.clashesOf(2))
	}
}

// Mods on together from before clashing mods were kept apart: the winner in
// the load order stays on.
func TestResolveClashesOnStartup(t *testing.T) {
	it := newInstallTest(t)

	for i, m := range []appdb.Mod{
		it.clashMod(t, 1, "First", true, "a.vmat_c"),
		// Loses to First.
		it.clashMod(t, 2, "Second", true, "a.vmat_c", "b.vmat_c"),
		// Clashes only with Second, which is going off, so it stays on.
		it.clashMod(t, 3, "Third", true, "b.vmat_c"),
		it.clashMod(t, 4, "Off", false, "a.vmat_c"),
	} {
		m.LoadOrder = i

		if _, err := it.db.AddMod(context.Background(), m); err != nil {
			t.Fatal(err)
		}
	}

	mainthread.Wait(func() {
		it.installedMods = newInstalledMods(it.client, it.db, it.store)
		it.ResolveClashes()
	})

	want := map[string]bool{"First": true, "Second": false, "Third": true, "Off": false}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after resolving: %v, want %v", got, want)
	}

	if got := it.message(); got != "Turned off Second, which changes the same files as another mod" {
		t.Errorf("message = %q", got)
	}

	if got := it.clashesOf(2); !strings.Contains(got, "First") || !strings.Contains(got, "Third") {
		t.Errorf("clashes of Second = %q", got)
	}

	mainthread.Wait(func() { it.Undo(it.count("messageId")) })

	if !it.enabled(t)["Second"] {
		t.Error("Undo didn't turn Second back on")
	}

	// With nothing clashing, there's nothing to say.
	before := it.count("messageId")

	mainthread.Wait(func() {
		it.SetEnabled(installedKey(banana.SectionMods, 2), false)
		it.ResolveClashes()
	})

	if it.count("messageId") != before {
		t.Errorf("message = %q, want none", it.message())
	}
}

// A VPK's file list is read from the store once, then kept: the store's
// files never change.
func TestVPKPathsReadOnce(t *testing.T) {
	it := newInstallTest(t)

	a := it.clashMod(t, 1, "A", true, "a.vmat_c")
	b := it.clashMod(t, 2, "B", false, "a.vmat_c", "b.vmat_c")
	addInstalled(t, it, a, b)

	for _, m := range []appdb.Mod{a, b} {
		if err := os.Remove(it.store.Path(m.Files[0].VPKs[0].SHA256)); err != nil {
			t.Fatal(err)
		}
	}

	mainthread.Wait(func() { it.updateClashes() })

	if got := it.clashesOf(1); got != "B" {
		t.Errorf("clashes of A after its VPK went = %q, want B", got)
	}
}

// A mod that's downloaded and installed starts on, so it turns off those it
// clashes with.
func TestInstallTurnsOffClashing(t *testing.T) {
	it := newInstallTest(t)

	old := it.clashMod(t, 1, "Old", true, "a.vmat_c")
	addInstalled(t, it, old)

	// An archive holding a VPK that changes the same file, and another.
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	w, _ := z.Create("pak01_dir.vpk")
	sources := []vpk.Source{}

	for _, p := range []string{"a.vmat_c", "b.vmat_c"} {
		data := []byte(p)
		sources = append(sources, vpk.Source{
			Path: p, CRC32: crc32.ChecksumIEEE(data), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
		})
	}

	if err := vpk.Write(w, sources); err != nil {
		t.Fatal(err)
	}

	z.Close()

	it.mu.Lock()
	it.files["/dl/new"] = archive.Bytes()
	it.mu.Unlock()

	sum := md5.Sum(archive.Bytes())
	file := banana.File{ID: 50, Name: "new.zip", Size: int64(archive.Len()), DownloadURL: it.srv.URL + "/dl/new",
		MD5: hex.EncodeToString(sum[:]), DateAdded: 1_790_000_000}

	info := testInfo(2)
	info.name = "New"
	key := installedKey(banana.SectionMods, 2)

	mainthread.Wait(func() { it.Install(info, []banana.File{file}) })
	it.waitForStatus(t, key, statusInstalled)

	want := map[string]bool{"Old": false, "New": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after installing New: %v, want %v", got, want)
	}

	if got := it.message(); got != "Turned off Old, which changes the same files" {
		t.Errorf("message = %q", got)
	}

	if got := it.clashesOf(1); got != "New" {
		t.Errorf("clashes of Old = %q", got)
	}
}
