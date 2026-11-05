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

	got := Clashes(mods)
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

	got := Clashes([]PathSet{cape, skin, other})
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Clashes = %v, want %v", got, want)
	}
}

func TestClashesNone(t *testing.T) {
	if got := Clashes(nil); len(got) != 0 {
		t.Errorf("Clashes(nil) = %v, want none", got)
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
