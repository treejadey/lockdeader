package deploy

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/treejadey/lockdeader/vpk"
)

// A PathSet is the files a VPK has, by path in lower case (the game doesn't
// care about case), with what each holds.
type PathSet map[string]Content

// Content tells files apart by what they hold, as a VPK's directory records
// it, without reading their data.
type Content struct {
	CRC32 uint32
	Size  uint32
}

func (s PathSet) overlaps(other PathSet) bool {
	for p := range other {
		if _, ok := s[p]; ok {
			return true
		}
	}

	return false
}

func (s PathSet) add(other PathSet) {
	for p, c := range other {
		s[p] = c
	}
}

func readPaths(f File) (PathSet, error) {
	return ReadPaths(f.Path)
}

// ReadPaths lists the files in the VPK at path, reading only its directory.
func ReadPaths(path string) (PathSet, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("deploy: %w", err)
	}

	defer r.Close()

	d, err := vpk.ReadDir(r)
	if err != nil {
		return nil, fmt.Errorf("deploy: %s: %w", path, err)
	}

	set := make(PathSet, len(d.Entries))

	for _, e := range d.Entries {
		set[strings.ToLower(e.Path)] = Content{e.CRC32, e.Size}
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
func allocate(files []File, paths func(File) (PathSet, error)) ([][]File, error) {
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

	sets := make([]PathSet, len(files))

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
func bundle(files []File, sets []PathSet) [][]File {
	var bundles [][]File
	var current []File
	used := PathSet{}

	for i, f := range files {
		if len(current) > 0 && used.overlaps(sets[i]) {
			bundles = append(bundles, current)
			current = nil
			used = PathSet{}
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
