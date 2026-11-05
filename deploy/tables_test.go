package deploy

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"io/fs"
	stdmaps "maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/treejadey/lockdeader/kv3"
	"github.com/treejadey/lockdeader/resource"
	"github.com/treejadey/lockdeader/vpk"
)

const pickTable = "soundevents/generated_vo_hero_pick_vo.vsndevts_c"

// soundTable compiles a table with one event per hero, playing the file
// given.
func soundTable(t *testing.T, heroes map[string]string) string {
	t.Helper()

	root := kv3.Value{Type: kv3.TypeObject}

	for _, hero := range slices.Sorted(stdmaps.Keys(heroes)) {
		root.Object = append(root.Object, kv3.Member{Key: hero, Value: kv3.Value{
			Type: kv3.TypeObject, Object: kv3.Object{{Key: "vsnd_files", Value: kv3.Value{Type: kv3.TypeString, String: heroes[hero]}}},
		}})
	}

	data, err := kv3.Encode(&kv3.Document{Format: kv3.FormatGeneric, Root: root})
	if err != nil {
		t.Fatal(err)
	}

	return string(resource.Write(&resource.Resource{Version: 1, Blocks: []resource.Block{{Type: "DATA", Data: data}}}))
}

func gameHeroes() map[string]string {
	return map[string]string{"Haze": "sounds/haze.vsnd", "Lash": "sounds/lash.vsnd", "Fencer": "sounds/fencer.vsnd"}
}

// voiceMod writes a VPK for a voice mod for one hero: its own sound, and
// its copy of the pick table with that hero's event playing it. stale
// changes another hero's event too, as an out-of-date copy does.
func voiceMod(t *testing.T, dir, hero, sound string, stale bool) Mod {
	t.Helper()

	heroes := gameHeroes()
	heroes[hero] = sound

	if stale {
		heroes["Fencer"] = "sounds/fencer_old.vsnd"
	}

	files := map[string]string{pickTable: soundTable(t, heroes), sound + "_c": hero}

	var sources []vpk.Source

	for p, data := range files {
		sources = append(sources, vpk.Source{
			Path: p, CRC32: crc32.ChecksumIEEE([]byte(data)), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader([]byte(data))), nil },
		})
	}

	var b bytes.Buffer
	if err := vpk.Write(&b, sources); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, hero+sound[len(sound)-6:]+".vpk")
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return Mod{Files: []File{{SHA256: hero, Path: path, Size: int64(b.Len())}}}
}

// mergedEvents reads the events of the pick table in a merged VPK.
func mergedEvents(t *testing.T, data []byte) map[string]string {
	t.Helper()

	d, err := vpk.ReadDir(bytes.NewReader(data))
	if err != nil || len(d.Entries) != 1 || d.Entries[0].Path != pickTable {
		t.Fatalf("merged VPK: %v, %+v; want just the pick table", err, d)
	}

	table, _ := io.ReadAll(d.Open(d.Entries[0]))

	r, err := resource.Read(table)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := r.Block("DATA")

	doc, err := kv3.Decode(block)
	if err != nil {
		t.Fatal(err)
	}

	events := map[string]string{}

	for _, e := range doc.Root.Object {
		events[e.Key] = e.Value.Object[0].Value.String
	}

	return events
}

func TestMergeTables(t *testing.T) {
	game, dir := t.TempDir(), t.TempDir()
	fakeBase(t, game, map[string]string{pickTable: soundTable(t, gameHeroes())})

	haze := voiceMod(t, dir, "Haze", "sounds/haze_new.vsnd", false)
	lash := voiceMod(t, dir, "Lash", "sounds/lash_new.vsnd", true)
	alone := Mod{Files: []File{{Path: filepath.Join(dir, "missing.vpk")}}}

	res, err := MergeTables(game, []Mod{haze, alone, lash})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"Haze": "sounds/haze_new.vsnd", "Lash": "sounds/lash_new.vsnd", "Fencer": "sounds/fencer.vsnd"}
	if got := mergedEvents(t, res.VPK); !stdmaps.Equal(got, want) {
		t.Errorf("merged events = %v, want %v", got, want)
	}

	if !slices.Equal(res.Tables, []string{pickTable}) {
		t.Errorf("tables = %v", res.Tables)
	}

	// Lash's copy's old Fencer event was left out; it shares the table with
	// Haze's mod.
	if len(res.LeftOut[0]) != 0 || len(res.LeftOut[1]) != 0 || !slices.Equal(res.LeftOut[2], []int{0}) {
		t.Errorf("left out = %v, want only the Lash mod's, beside Haze's", res.LeftOut)
	}

	// The same mods give the same VPK, so an apply that changes nothing
	// rewrites nothing in the game.
	again, _ := MergeTables(game, []Mod{haze, alone, lash})
	if !bytes.Equal(again.VPK, res.VPK) {
		t.Error("merging the same mods again made a different VPK")
	}
}

func TestMergeTablesNothingToMerge(t *testing.T) {
	game, dir := t.TempDir(), t.TempDir()
	fakeBase(t, game, map[string]string{pickTable: soundTable(t, gameHeroes())})

	haze := voiceMod(t, dir, "Haze", "sounds/haze_new.vsnd", false)

	// One mod with the table, or two with the same copy of it.
	for _, mods := range [][]Mod{{haze}, {haze, haze}} {
		res, err := MergeTables(game, mods)
		if err != nil || res.VPK != nil {
			t.Errorf("MergeTables of %d mods = %v, %v; want nothing", len(mods), res.VPK, err)
		}
	}

	// Without the game, nothing is merged, and the error says so.
	lash := voiceMod(t, dir, "Lash", "sounds/lash_new.vsnd", false)

	res, err := MergeTables(t.TempDir(), []Mod{haze, lash})
	if err == nil || !errors.Is(err, fs.ErrNotExist) || res.VPK != nil {
		t.Errorf("MergeTables without a game = %v, %v; want nothing and an error", res.VPK, err)
	}
}
