package deploy

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/treejadey/lockdeader/vpk"
)

// pathSet is the files a VPK has, by path in lower case: the game doesn't
// care about case.
type pathSet map[string]struct{}

func (s pathSet) overlaps(other pathSet) bool {
	for p := range other {
		if _, ok := s[p]; ok {
			return true
		}
	}

	return false
}

func (s pathSet) add(other pathSet) {
	for p := range other {
		s[p] = struct{}{}
	}
}

// readPaths lists the files in the VPK at f.Path, reading only its directory.
func readPaths(f File) (pathSet, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return nil, fmt.Errorf("deploy: %w", err)
	}

	defer r.Close()

	d, err := vpk.ReadDir(r)
	if err != nil {
		return nil, fmt.Errorf("deploy: %s: %w", f.SHA256, err)
	}

	set := make(pathSet, len(d.Entries))

	for _, e := range d.Entries {
		set[strings.ToLower(e.Path)] = struct{}{}
	}

	return set, nil
}

// allocate shares the game's 99 slots out among files, in load order: each
// slot is one or more of them. Up to 99 files get a slot each. Past that,
// the files lowest in the load order are bundled into as few VPKs as will
// fit, since VPKs that don't change any of the same files can share one.
// Two that do are never bundled together: which wins then stays plain from
// the slot numbers, so the bundles are split wherever a file would overlap
// one already in the bundle. paths says which files a VPK has; it's only
// called when there's bundling to do.
func allocate(files []File, paths func(File) (pathSet, error)) ([][]File, error) {
	singles := func(files []File) [][]File {
		slots := make([][]File, len(files))

		for i := range files {
			slots[i] = files[i : i+1]
		}

		return slots
	}

	if len(files) <= Slots {
		return singles(files), nil
	}

	sets := make([]pathSet, len(files))

	for i, f := range files {
		set, err := paths(f)
		if err != nil {
			return nil, err
		}

		sets[i] = set
	}

	// Bundle as few files as possible, from the bottom of the load order.
	for head := Slots - 1; head >= 0; head-- {
		bundles := bundle(files[head:], sets[head:])

		if head+len(bundles) <= Slots {
			return append(singles(files[:head]), bundles...), nil
		}
	}

	return nil, ErrTooMany
}

// bundle groups consecutive files, starting a new group whenever a file
// overlaps one already in the group.
func bundle(files []File, sets []pathSet) [][]File {
	var bundles [][]File
	var current []File
	used := pathSet{}

	for i, f := range files {
		if len(current) > 0 && used.overlaps(sets[i]) {
			bundles = append(bundles, current)
			current = nil
			used = pathSet{}
		}

		current = append(current, f)
		used.add(sets[i])
	}

	if len(current) > 0 {
		bundles = append(bundles, current)
	}

	return bundles
}

// writeBundle writes one VPK holding every file of the slot's members to
// path, which mustn't exist yet, and returns its size.
func writeBundle(path string, slot []File) (int64, error) {
	var sources []vpk.Source

	for _, f := range slot {
		r, err := os.Open(f.Path)
		if err != nil {
			return 0, fmt.Errorf("deploy: %w", err)
		}

		defer r.Close()

		d, err := vpk.ReadDir(r)
		if err != nil {
			return 0, fmt.Errorf("deploy: %s: %w", f.SHA256, err)
		}

		for _, e := range d.Entries {
			sources = append(sources, vpk.Source{
				Path: e.Path, CRC32: e.CRC32, Size: e.Size,
				Open: func() (io.ReadCloser, error) { return io.NopCloser(d.Open(e)), nil },
			})
		}
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, fmt.Errorf("deploy: %w", err)
	}

	if err := vpk.Write(out, sources); err != nil {
		out.Close()

		return 0, fmt.Errorf("deploy: %w", err)
	}

	if err := out.Sync(); err != nil {
		out.Close()

		return 0, fmt.Errorf("deploy: %w", err)
	}

	if err := out.Close(); err != nil {
		return 0, fmt.Errorf("deploy: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("deploy: %w", err)
	}

	return info.Size(), nil
}
