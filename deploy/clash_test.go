package deploy

import (
	"os"
	"slices"
	"testing"
)

func paths(list ...string) PathSet {
	set := make(PathSet, len(list))

	for _, p := range list {
		set[p] = struct{}{}
	}

	return set
}

func TestClashes(t *testing.T) {
	mods := []PathSet{
		// Two skins for the same hero, which share one file of many.
		paths("models/heroes/abrams/abrams.vmdl_c", "materials/abrams/body.vmat_c"),
		paths("models/heroes/abrams/abrams.vmdl_c", "materials/abrams/gold.vmat_c"),
		// Doesn't touch anything the others do.
		paths("sounds/ui/click.vsnd_c"),
		// Shares only the files that don't count.
		paths("addoninfo.txt", "readme.txt", "panorama/image_compiler.vdata_c",
			"materials/default/default_color.vtex_c", "sounds/ui/hover.vsnd_c"),
		paths("addoninfo.txt", "modinfo.json", "readme_install.md", "panorama/image_compiler.vdata_c",
			"materials/default/default_color.vtex_c", "sounds/ui/select.vsnd_c"),
		// Clashes with both skins.
		paths("models/heroes/abrams/abrams.vmdl_c"),
		// No files at all, e.g. one that couldn't be read.
		nil,
	}

	want := [][]int{{1, 5}, {0, 5}, nil, nil, nil, {0, 1}, nil}

	got := Clashes(mods)
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

	want := paths("models/heroes/abrams.vmdl_c", "sounds/x.vsnd_c")
	if len(got) != len(want) {
		t.Fatalf("ReadPaths = %v, want %v", got, want)
	}

	for p := range want {
		if _, ok := got[p]; !ok {
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
