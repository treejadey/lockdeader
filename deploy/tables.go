package deploy

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/treejadey/lockdeader/merge"
	"github.com/treejadey/lockdeader/vpk"
)

// A Mod is a mod that's on: its VPKs, in load order.
type Mod struct {
	Files []File
}

// Merged is what MergeTables made.
type Merged struct {
	// A VPK holding the merged tables, to load before every mod, so it wins
	// them; nil when no table needed merging.
	VPK []byte
	// Tables merged, by path.
	Tables []string
	// For each mod: the other mods it shares a table with in which the
	// merge left some of its events out. Empty when nothing of it was.
	LeftOut [][]int
}

// MergeTables merges the sound event tables that two or more of mods, given
// in load order, have different copies of, over the base game's copies in
// gameDir (see merge.SoundEvents). A table only one mod has, or that every
// mod has the same copy of, is left as the mods have it.
//
// It does what it can: a table that can't be merged, or a VPK that can't be
// read, is left out, and the error says why, beside a usable result. Files
// that aren't there are left for Apply to report.
func MergeTables(gameDir string, mods []Mod) (*Merged, error) {
	res := &Merged{LeftOut: make([][]int, len(mods))}

	var errs []error

	// Which mods have each table, and every file each mod has, which tells
	// which events use its own sounds.
	sets := make([]PathSet, len(mods))
	owners := map[string][]int{}

	for i, m := range mods {
		sets[i] = PathSet{}

		for _, f := range m.Files {
			set, err := ReadPaths(f.Path)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					errs = append(errs, err)
				}

				continue
			}

			for p, c := range set {
				if _, ok := sets[i][p]; !ok {
					sets[i][p] = c
				}
			}
		}

		for p := range sets[i] {
			if strings.HasSuffix(p, ".vsndevts_c") {
				owners[p] = append(owners[p], i)
			}
		}
	}

	var tables []string

	for p, ids := range owners {
		if len(ids) > 1 && slices.ContainsFunc(ids, func(i int) bool { return sets[i][p] != sets[ids[0]][p] }) {
			tables = append(tables, p)
		}
	}

	if len(tables) == 0 {
		return res, errors.Join(errs...)
	}

	slices.Sort(tables)

	base, err := ReadBase(gameDir)
	if err != nil {
		return res, errors.Join(append(errs, fmt.Errorf("deploy: tables can't be merged without the game's: %w", err))...)
	}

	var sources []vpk.Source

	for _, table := range tables {
		data, leftOut, err := mergeTable(base, table, owners[table], mods, sets)
		if err != nil {
			errs = append(errs, fmt.Errorf("deploy: %s isn't merged: %w", table, err))

			continue
		}

		res.Tables = append(res.Tables, table)
		sources = append(sources, vpk.Source{
			Path: table, CRC32: crc32.ChecksumIEEE(data), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
		})

		for k, i := range owners[table] {
			if len(leftOut[k]) == 0 {
				continue
			}

			for _, j := range owners[table] {
				if j != i && !slices.Contains(res.LeftOut[i], j) {
					res.LeftOut[i] = append(res.LeftOut[i], j)
				}
			}
		}
	}

	for i := range res.LeftOut {
		slices.Sort(res.LeftOut[i])
	}

	if len(sources) > 0 {
		var b bytes.Buffer

		if err := vpk.Write(&b, sources); err != nil {
			return &Merged{LeftOut: make([][]int, len(mods))}, errors.Join(append(errs, fmt.Errorf("deploy: %w", err))...)
		}

		res.VPK = b.Bytes()
	}

	return res, errors.Join(errs...)
}

// mergeTable merges the copies of a table that the mods ids have.
func mergeTable(base *Base, table string, ids []int, mods []Mod, sets []PathSet) ([]byte, [][]string, error) {
	game, _, err := base.File(table)
	if err != nil {
		return nil, nil, err
	}

	copies := make([]merge.Copy, len(ids))

	for k, i := range ids {
		data, err := mods[i].ReadFile(table)
		if err != nil {
			return nil, nil, err
		}

		copies[k] = merge.Copy{Data: data, Ships: func(p string) bool { _, ok := sets[i][p]; return ok }}
	}

	res, err := merge.SoundEvents(game, copies)
	if err != nil {
		return nil, nil, err
	}

	return res.Data, res.LeftOut, nil
}

// ReadFile reads a file, by its path in lower case, from the first of the
// mod's VPKs that has it.
func (m Mod) ReadFile(path string) ([]byte, error) {
	for _, f := range m.Files {
		r, err := os.Open(f.Path)
		if err != nil {
			continue
		}

		d, err := vpk.ReadDir(r)

		// ReadFile opens the file itself.
		r.Close()

		if err != nil {
			continue
		}

		for _, e := range d.Entries {
			if strings.ToLower(e.Path) == path {
				return d.ReadFile(e, f.Path)
			}
		}
	}

	return nil, fmt.Errorf("no VPK of the mod has %s", path)
}
