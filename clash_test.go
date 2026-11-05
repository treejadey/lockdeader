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
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/kv3"
	"github.com/treejadey/lockdeader/resource"
	"github.com/treejadey/lockdeader/vpk"
)

// clashMod is a mod with one VPK holding paths, already in the store, as
// AddInstalled and the database take it.
func (it *installTest) clashMod(t *testing.T, id int, name string, enabled bool, paths ...string) appdb.Mod {
	t.Helper()

	// Each mod has its own copy of its files: ones exactly the same as
	// another's don't clash.
	files := make(map[string][]byte, len(paths))

	for _, p := range paths {
		files[p] = []byte(name + ":" + p)
	}

	return it.modWithFiles(t, id, name, enabled, files)
}

// modWithFiles is a mod with one VPK holding files, already in the store.
func (it *installTest) modWithFiles(t *testing.T, id int, name string, enabled bool, files map[string][]byte) appdb.Mod {
	t.Helper()

	var b bytes.Buffer
	if err := vpk.Write(&b, vpkSources(files)); err != nil {
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

func vpkSources(files map[string][]byte) []vpk.Source {
	var sources []vpk.Source

	for p, data := range files {
		sources = append(sources, vpk.Source{
			Path: p, CRC32: crc32.ChecksumIEEE(data), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
		})
	}

	return sources
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

func TestIdenticalCopiesDontClash(t *testing.T) {
	it := newInstallTest(t)

	one := it.clashMod(t, 1, "One", true, "one.vmat_c")
	two := it.clashMod(t, 2, "Two", true, "two.vmat_c")

	// Both made with the same tool, which puts the same note in each.
	note := it.clashMod(t, 3, "Note", false, "deadlockforge/made_with_dlf.txt")
	one.Files = append(one.Files, appdb.ModFile{GameBananaID: 11, Name: "note.zip", VPKs: note.Files[0].VPKs})
	two.Files = append(two.Files, appdb.ModFile{GameBananaID: 21, Name: "note.zip", VPKs: note.Files[0].VPKs})

	addInstalled(t, it, one, two)

	want := map[string]bool{"One": true, "Two": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("mods sharing only identical files: %v, want %v", got, want)
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
		it.installedMods = newInstalledMods(it.client, it.db, it.store, nil)
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
		data := []byte("New:" + p)
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

// soundTable compiles a sound event table: one event per hero, each
// playing the file given.
func soundTable(t *testing.T, heroes map[string]string) []byte {
	t.Helper()

	root := kv3.Value{Type: kv3.TypeObject}

	for _, hero := range slices.Sorted(maps.Keys(heroes)) {
		root.Object = append(root.Object, kv3.Member{Key: "Generated." + hero + ".Hero.Pick.VO", Value: kv3.Value{
			Type: kv3.TypeObject, Object: kv3.Object{{Key: "vsnd_files", Value: kv3.Value{
				Type: kv3.TypeString, String: heroes[hero]}}},
		}})
	}

	data, err := kv3.Encode(&kv3.Document{Format: kv3.FormatGeneric, Root: root})
	if err != nil {
		t.Fatal(err)
	}

	return resource.Write(&resource.Resource{Version: 1, Blocks: []resource.Block{{Type: "DATA", Data: data}}})
}

const pickTable = "soundevents/generated_vo_hero_pick_vo.vsndevts_c"

// fakeGame makes a game folder whose pak01_dir.vpk has the game's pick
// table, and has the mods merge over it.
func (it *installTest) fakeGame(t *testing.T) {
	t.Helper()

	dir := writeFakeGame(t)

	mainthread.Wait(func() { it.gameFolder = func() string { return dir } })
}

func writeFakeGame(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	pak := deploy.BasePak(dir)

	var b bytes.Buffer
	if err := vpk.Write(&b, vpkSources(map[string][]byte{pickTable: soundTable(t, map[string]string{
		"Haze": "sounds/vo/haze/select.vsnd", "Lash": "sounds/vo/lash/select.vsnd", "Fencer": "sounds/vo/fencer/select.vsnd",
	})})); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(pak), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(pak, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// voiceMod is a voice mod for one hero: its own sound, and its copy of the
// pick table pointing that hero's event at it.
func (it *installTest) voiceMod(t *testing.T, id int, name string, enabled bool, hero, sound string) appdb.Mod {
	t.Helper()

	table := map[string]string{"Haze": "sounds/vo/haze/select.vsnd", "Lash": "sounds/vo/lash/select.vsnd", "Fencer": "sounds/vo/fencer/select.vsnd"}
	table[hero] = sound

	return it.modWithFiles(t, id, name, enabled, map[string][]byte{
		pickTable:    soundTable(t, table),
		sound + "_c": []byte(name),
	})
}

// Voice mods for different heroes share the pick table, but merge rather
// than clash.
func TestSharedTablesMerge(t *testing.T) {
	it := newInstallTest(t)
	it.fakeGame(t)

	haze := it.voiceMod(t, 1, "Haze Voice", true, "Haze", "sounds/vo/haze/new.vsnd")
	lash := it.voiceMod(t, 2, "Lash Voice", false, "Lash", "sounds/vo/lash/new.vsnd")
	alsoHaze := it.voiceMod(t, 3, "Other Haze Voice", false, "Haze", "sounds/vo/haze/other.vsnd")
	addInstalled(t, it, haze, lash, alsoHaze)

	mainthread.Wait(func() { it.SetEnabled(installedKey(banana.SectionMods, 2), true) })

	want := map[string]bool{"Haze Voice": true, "Lash Voice": true, "Other Haze Voice": false}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after turning on Lash Voice: %v, want %v", got, want)
	}

	if it.clashesOf(1) != "Other Haze Voice" || it.clashesOf(2) != "" {
		t.Errorf("clashes = %q and %q, want only Haze's with the other Haze voice", it.clashesOf(1), it.clashesOf(2))
	}

	// Both change Haze's event with their own sounds: a real clash.
	mainthread.Wait(func() { it.SetEnabled(installedKey(banana.SectionMods, 3), true) })

	want = map[string]bool{"Haze Voice": false, "Lash Voice": true, "Other Haze Voice": true}
	if got := it.enabled(t); !maps.Equal(got, want) {
		t.Errorf("after turning on Other Haze Voice: %v, want %v", got, want)
	}
}

// Without the game's copy to merge over, mods that share a table clash.
func TestSharedTablesClashWithoutGame(t *testing.T) {
	it := newInstallTest(t)

	addInstalled(t, it,
		it.voiceMod(t, 1, "Haze Voice", true, "Haze", "sounds/vo/haze/new.vsnd"),
		it.voiceMod(t, 2, "Lash Voice", false, "Lash", "sounds/vo/lash/new.vsnd"))

	if got := it.clashesOf(2); got != "Haze Voice" {
		t.Errorf("clashes of Lash Voice without a game = %q, want Haze Voice", got)
	}

	// Once the game is known, they merge.
	it.fakeGame(t)
	mainthread.Wait(func() { it.updateClashes() })

	if got := it.clashesOf(2); got != "" {
		t.Errorf("clashes of Lash Voice with the game = %q, want none", got)
	}
}

// The marker on a mod the merge left changes out of names the mods it's
// merged with.
func TestSetMerged(t *testing.T) {
	it := newInstallTest(t)

	addInstalled(t, it,
		it.clashMod(t, 1, "Haze Voice", true, "a.vsnd_c"),
		it.clashMod(t, 2, "Lash Voice", true, "b.vsnd_c"),
		it.clashMod(t, 3, "Flowery", true, "c.vsnd_c"))

	mergedWith := func(id int) string {
		return it.role(installedKey(banana.SectionMods, id), roleInstalledMergedWith).ToString()
	}

	mainthread.Wait(func() { it.SetMerged(map[string][]string{"Mod/3": {"Mod/1", "Mod/2"}}) })

	if mergedWith(3) != "Haze Voice, Lash Voice" || mergedWith(1) != "" || mergedWith(2) != "" {
		t.Errorf("merged with = %q, %q, %q; want only Flowery's", mergedWith(1), mergedWith(2), mergedWith(3))
	}

	// Without a game, nothing is merged.
	mainthread.Wait(func() { it.SetMerged(nil) })

	if mergedWith(3) != "" {
		t.Errorf("merged with after SetMerged(nil) = %q", mergedWith(3))
	}
}

// On startup, mods that share a table merge from the start: the clash
// markers are worked out with the game folder, not before it's known.
func TestSharedTablesMergeOnStartup(t *testing.T) {
	it := newInstallTest(t)
	game := writeFakeGame(t)

	for _, m := range []appdb.Mod{
		it.voiceMod(t, 1, "Haze Voice", true, "Haze", "sounds/vo/haze/new.vsnd"),
		it.voiceMod(t, 2, "Lash Voice", true, "Lash", "sounds/vo/lash/new.vsnd"),
	} {
		if _, err := it.db.AddMod(context.Background(), m); err != nil {
			t.Fatal(err)
		}
	}

	mainthread.Wait(func() {
		it.installedMods = newInstalledMods(it.client, it.db, it.store, func() string { return game })
	})

	if it.clashesOf(1) != "" || it.clashesOf(2) != "" {
		t.Errorf("clashes on startup = %q and %q, want none", it.clashesOf(1), it.clashesOf(2))
	}
}
