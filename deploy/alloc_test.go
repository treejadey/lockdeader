package deploy

import (
	"errors"
	"fmt"
	"testing"
)

// fakeFiles makes n files, each with its own path, plus the shared ones
// given: shared maps a file's index to extra paths.
func fakeFiles(n int, shared map[int][]string) ([]File, func(File) (pathSet, error)) {
	files := make([]File, n)
	sets := make(map[string]pathSet, n)

	for i := range files {
		files[i] = File{SHA256: fmt.Sprintf("f%d", i)}
		set := pathSet{fmt.Sprintf("own/%d", i): {}}

		for _, p := range shared[i] {
			set[p] = struct{}{}
		}

		sets[files[i].SHA256] = set
	}

	return files, func(f File) (pathSet, error) { return sets[f.SHA256], nil }
}

func sizes(slots [][]File) []int {
	var out []int

	for _, s := range slots {
		out = append(out, len(s))
	}

	return out
}

func TestAllocateSingles(t *testing.T) {
	files, _ := fakeFiles(99, nil)

	slots, err := allocate(files, func(File) (pathSet, error) {
		t.Error("read a VPK with nothing to bundle")

		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(slots) != 99 {
		t.Errorf("got %d slots", len(slots))
	}

	for i, s := range slots {
		if len(s) != 1 || s[0].SHA256 != files[i].SHA256 {
			t.Errorf("slot %d = %v", i+1, s)
		}
	}
}

func TestAllocateBundlesTheTail(t *testing.T) {
	tests := []struct {
		n       int
		singles int
		bundles []int
	}{
		{100, 98, []int{2}},
		{150, 98, []int{52}},
		{500, 98, []int{402}},
	}

	for _, tc := range tests {
		files, paths := fakeFiles(tc.n, nil)

		slots, err := allocate(files, paths)
		if err != nil {
			t.Fatal(err)
		}

		got := sizes(slots)
		want := append(make([]int, 0, Slots), tc.bundles...)

		for range tc.singles {
			want = append([]int{1}, want...)
		}

		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%d files: slots %v, want %v", tc.n, got, want)
		}

		// The lowest in the load order are the ones bundled.
		last := slots[len(slots)-1]
		if last[len(last)-1].SHA256 != files[tc.n-1].SHA256 {
			t.Errorf("%d files: the last slot doesn't end with the last file", tc.n)
		}
	}
}

func TestAllocateSplitsBundlesAtOverlaps(t *testing.T) {
	// Files 99 and 100 change the same file, so they can't share a slot.
	files, paths := fakeFiles(120, map[int][]string{99: {"shared"}, 100: {"shared"}})

	slots, err := allocate(files, paths)
	if err != nil {
		t.Fatal(err)
	}

	if len(slots) != 99 {
		t.Fatalf("got %d slots", len(slots))
	}

	// 97 singles, then {97, 98, 99} and {100 … 119}.
	if got := sizes(slots)[96:]; fmt.Sprint(got) != "[1 3 20]" {
		t.Errorf("slots end %v", got)
	}

	slotOf := func(sha string) int {
		for i, s := range slots {
			for _, f := range s {
				if f.SHA256 == sha {
					return i
				}
			}
		}

		return -1
	}

	if slotOf("f99") >= slotOf("f100") {
		t.Error("the earlier of two overlapping files doesn't come first")
	}
}

func TestAllocateTooMany(t *testing.T) {
	shared := map[int][]string{}

	for i := range 101 {
		shared[i] = []string{"everyone/has/this"}
	}

	files, paths := fakeFiles(101, shared)

	if _, err := allocate(files, paths); !errors.Is(err, ErrTooMany) {
		t.Errorf("err = %v, want ErrTooMany", err)
	}
}

func TestAllocatePathsError(t *testing.T) {
	files, _ := fakeFiles(100, nil)
	boom := errors.New("boom")

	if _, err := allocate(files, func(File) (pathSet, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}
