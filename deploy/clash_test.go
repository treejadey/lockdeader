package deploy

import (
	"hash/crc32"
	"os"
	"slices"
	"testing"
)

// modFiles is the files of a mod whose copies of them are its own: no other
// mod made with a different n has the same.
func modFiles(n uint32, list ...string) PathSet {
	set := make(PathSet, len(list))

	for _, p := range list {
		set[p] = Content{CRC32: n, Size: 100}
	}

	return set
}

func TestClashes(t *testing.T) {
	mods := []PathSet{
		// Two skins for the same hero, which share one file of many.
		modFiles(1, "models/heroes/abrams/abrams.vmdl_c", "materials/abrams/body.vmat_c"),
		modFiles(2, "models/heroes/abrams/abrams.vmdl_c", "materials/abrams/gold.vmat_c"),
		// Doesn't touch anything the others do.
		modFiles(3, "sounds/ui/click.vsnd_c"),
		// Shares only the files that don't count.
		modFiles(4, "addoninfo.txt", "readme.txt", "panorama/image_compiler.vdata_c",
			"materials/default/default_color.vtex_c", "sounds/ui/hover.vsnd_c"),
		modFiles(5, "addoninfo.txt", "modinfo.json", "readme_install.md", "panorama/image_compiler.vdata_c",
			"materials/default/default_color.vtex_c", "sounds/ui/select.vsnd_c"),
		// Clashes with both skins.
		modFiles(6, "models/heroes/abrams/abrams.vmdl_c"),
		// No files at all, e.g. one that couldn't be read.
		nil,
	}

	want := [][]int{{1, 5}, {0, 5}, nil, nil, nil, {0, 1}, nil}

	got := Clashes(mods, nil)
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Clashes = %v, want %v", got, want)
	}
}

func TestClashesIdenticalCopies(t *testing.T) {
	// Both made with DeadlockForge, which puts the same note in every VPK.
	note := PathSet{"deadlockforge/made_with_dlf.txt": {CRC32: 0x19cc7249, Size: 50}}

	cape := modFiles(1, "materials/ratking/cape.vtex_c")
	skin := modFiles(2, "models/heroes/deadpack/deadpack.vmdl_c")
	cape.add(note)
	skin.add(note)

	// The same file again, but a different copy of it.
	other := modFiles(3, "deadlockforge/made_with_dlf.txt")

	want := [][]int{{2}, {2}, {0, 1}}

	got := Clashes([]PathSet{cape, skin, other}, nil)
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Clashes = %v, want %v", got, want)
	}
}

func TestClashesNone(t *testing.T) {
	if got := Clashes(nil, nil); len(got) != 0 {
		t.Errorf("Clashes(nil, nil) = %v, want none", got)
	}
}

func TestReadPathsLowerCase(t *testing.T) {
	s := newStore(t)
	f := s.add(map[string]string{"Models/Heroes/Abrams.vmdl_c": "a", "sounds/x.vsnd_c": "b"})

	got, err := ReadPaths(f.Path)
	if err != nil {
		t.Fatal(err)
	}

	want := PathSet{
		"models/heroes/abrams.vmdl_c": {CRC32: crc32.ChecksumIEEE([]byte("a")), Size: 1},
		"sounds/x.vsnd_c":             {CRC32: crc32.ChecksumIEEE([]byte("b")), Size: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("ReadPaths = %v, want %v", got, want)
	}

	for p, c := range want {
		if got[p] != c {
			t.Errorf("ReadPaths = %v, want %v", got, want)
		}
	}
}

func TestReadPathsVersion1(t *testing.T) {
	s := newStore(t)
	f := s.add(map[string]string{"sounds/x.vsnd_c": "b"})
	f = s.put(version1(t, readFile(t, f.Path)))

	got, err := ReadPaths(f.Path)
	if err != nil {
		t.Fatal(err)
	}

	if want := (Content{CRC32: crc32.ChecksumIEEE([]byte("b")), Size: 1}); len(got) != 1 || got["sounds/x.vsnd_c"] != want {
		t.Errorf("ReadPaths = %v", got)
	}
}

func TestReadPathsNotAVPK(t *testing.T) {
	s := newStore(t)
	f := s.add(map[string]string{"a.txt": "a"})

	// Cut off after its header, so the directory can't be read.
	if err := os.Truncate(f.Path, 12); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadPaths(f.Path); err == nil {
		t.Error("ReadPaths read a broken VPK")
	}
}

func TestClashesMergeable(t *testing.T) {
	table := "soundevents/hero/haze.vsndevts_c"
	mods := []PathSet{
		modFiles(1, table, "sounds/a.vsnd_c"),
		modFiles(2, table, "sounds/b.vsnd_c"),
		modFiles(3, table, "models/c.vmdl_c"),
		// Also changes a model the third does: still a clash.
		modFiles(4, "models/c.vmdl_c"),
	}

	var asked [][2]int

	// The first two copies of the table merge; the third's conflicts with
	// both. The model isn't a table, so it can't merge.
	mergeable := func(path string, a, b int) bool {
		asked = append(asked, [2]int{a, b})

		return path == table && b < 2
	}

	want := [][]int{{2}, {2}, {0, 1, 3}, {2}}

	if got := Clashes(mods, mergeable); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Clashes = %v, want %v", got, want)
	}

	for _, pair := range asked {
		if pair[0] >= pair[1] {
			t.Errorf("asked about %v, want the earlier mod first", pair)
		}
	}
}
