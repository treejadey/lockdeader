// Package deploy puts the mods that are turned on into Deadlock. It keeps a
// folder of its own in the game, game/citadel/lockdeader, holding exactly the
// VPKs the game should load, and adds or removes Lockdeader's own lines in
// gameinfo.gi, which make the game load that folder. See
// docs/mod-support-plan.md.
//
// Nothing in the game folder is the only copy of anything: it's all rebuilt
// from the store by Apply, so a crash at any point leaves the game loading
// an older or partial set of mods until the next Apply, never broken.
package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Slots is how many VPKs the game mounts from one search path:
// pak01_dir.vpk to pak99_dir.vpk.
const Slots = 99

// SearchPath is our folder as gameinfo.gi names it, relative to game/.
const SearchPath = "citadel/lockdeader"

const (
	manifestName = "lockdeader.json"
	tempSuffix   = ".tmp"
)

var (
	// Only possible when 100 or more mods that are on change the same files
	// as one another; see allocate.
	ErrTooMany = errors.New("deploy: too many of the mods that are on change the same files to fit in 99 slots")
	// Some files aren't in the store; Apply puts in the rest.
	ErrMissing     = errors.New("deploy: a mod's files are missing")
	ErrGameRunning = errors.New("deploy: Deadlock is running")
	// gameinfo.gi can't be understood: it has no SearchPaths block with a
	// "Game citadel" line.
	ErrNoSearchPaths = errors.New("deploy: gameinfo.gi has no SearchPaths block with a Game citadel line")
)

// A File is a VPK the game should load. Apply takes them in load order: the
// first wins any file two of them share.
type File struct {
	SHA256 string // its name in the store, and in the manifest
	Path   string // where it is in the store
	Size   int64
}

type Result struct {
	Placed  int      // slots written: a mod's VPK, or a bundle of several
	Removed int      // files deleted, leftovers included
	Bundled int      // VPKs that went into bundles rather than slots of their own
	Missing []string // SHA-256s of files that aren't in the store, so weren't placed
}

// Dir is our folder inside the game folder.
func Dir(gameDir string) string {
	return filepath.Join(gameDir, "game", "citadel", "lockdeader")
}

func slotName(i int) string {
	return fmt.Sprintf("pak%02d_dir.vpk", i)
}

// The manifest, lockdeader.json in our folder, says what each slot holds,
// so Apply can tell what's already right without hashing anything. It's
// deleted before the folder is changed and written again afterwards, so a
// crash in between means "check everything" next time: the manifest never
// claims more than is true.
type manifest struct {
	Slots map[string]manifestSlot `json:"slots"`
}

type manifestSlot struct {
	Members []string `json:"members"` // the SHA-256s of the VPKs in it, in order
	Size    int64    `json:"size"`
}

func readManifest(dir string) manifest {
	var m manifest

	data, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil || json.Unmarshal(data, &m) != nil || m.Slots == nil {
		return manifest{Slots: map[string]manifestSlot{}}
	}

	return m
}

func writeManifest(dir string, m manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return writeFileAtomic(filepath.Join(dir, manifestName), data, 0o644)
}

// Apply makes Dir(gameDir) hold files, as pak01_dir.vpk, pak02_dir.vpk and
// so on in that order, and nothing else. Mods that don't fit the 99 slots
// are bundled into fewer VPKs (see allocate). It touches nothing outside
// that folder. Applying the same files again does nothing.
//
// It must not run while Deadlock is (see Running); it doesn't check that
// itself. A failure part way leaves the folder with a mix of old and new
// files, which the game can still start with; the next Apply carries on.
func Apply(gameDir string, files []File) (Result, error) {
	var res Result
	var errs []error

	present := make([]File, 0, len(files))

	for _, f := range files {
		if _, err := os.Stat(f.Path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				res.Missing = append(res.Missing, f.SHA256)

				continue
			}

			return res, fmt.Errorf("deploy: %w", err)
		}

		present = append(present, f)
	}

	if len(res.Missing) > 0 {
		errs = append(errs, fmt.Errorf("%w: %s", ErrMissing, strings.Join(res.Missing, ", ")))
	}

	slots, err := allocate(present, readPaths)
	if err != nil {
		return res, err
	}

	dir := Dir(gameDir)

	// Mkdir, not MkdirAll: if game/citadel is gone, this must fail rather
	// than make one.
	if err := os.Mkdir(dir, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return res, fmt.Errorf("deploy: %w", err)
	}

	old := readManifest(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return res, fmt.Errorf("deploy: %w", err)
	}

	existing := make(map[string]fs.DirEntry, len(entries))

	for _, e := range entries {
		existing[e.Name()] = e
	}

	// Work out which slots are already right, and which need placing.
	next := manifest{Slots: make(map[string]manifestSlot, len(slots))}
	wanted := make(map[string]bool, len(slots))
	var toPlace []int

	for i, slot := range slots {
		name := slotName(i + 1)
		wanted[name] = true

		if m, ok := old.Slots[name]; ok && slices.Equal(m.Members, members(slot)) && hasSize(existing[name], m.Size) {
			next.Slots[name] = m

			continue
		}

		toPlace = append(toPlace, i)
	}

	// Everything in the folder is ours, so anything not wanted goes.
	var extras []string

	for name := range existing {
		if !wanted[name] && name != manifestName {
			extras = append(extras, name)
		}
	}

	slices.Sort(extras)

	if len(toPlace) == 0 && len(extras) == 0 && len(old.Slots) == len(slots) {
		return res, errors.Join(errs...)
	}

	if err := removeFile(filepath.Join(dir, manifestName)); err != nil {
		return res, err
	}

	for _, name := range extras {
		if err := removeEntry(filepath.Join(dir, name)); err != nil {
			return res, err
		}

		res.Removed++
	}

	var placeErr error

	for _, i := range toPlace {
		name := slotName(i + 1)

		size, err := placeSlot(dir, name, slots[i])
		if err != nil {
			placeErr = err

			break
		}

		next.Slots[name] = manifestSlot{Members: members(slots[i]), Size: size}
		res.Placed++

		if len(slots[i]) > 1 {
			res.Bundled += len(slots[i])
		}
	}

	// Written whatever happened, so it says what's really there.
	if err := writeManifest(dir, next); err != nil {
		errs = append(errs, err)
	}

	if placeErr != nil {
		errs = append([]error{placeErr}, errs...)
	}

	return res, errors.Join(errs...)
}

func members(slot []File) []string {
	shas := make([]string, len(slot))

	for i, f := range slot {
		shas[i] = f.SHA256
	}

	return shas
}

func hasSize(e fs.DirEntry, size int64) bool {
	if e == nil || !e.Type().IsRegular() {
		return false
	}

	info, err := e.Info()

	return err == nil && info.Size() == size
}

// placeSlot writes a slot under a temporary name, then renames it into
// place, so the slot is never half written.
func placeSlot(dir, name string, slot []File) (size int64, err error) {
	tmp := filepath.Join(dir, name+tempSuffix)
	dst := filepath.Join(dir, name)

	if err := removeFile(tmp); err != nil {
		return 0, err
	}

	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()

	if len(slot) == 1 {
		size = slot[0].Size
		err = placeFile(slot[0].Path, tmp, size)
	} else {
		size, err = writeBundle(tmp, slot)
	}

	if err != nil {
		return 0, err
	}

	if err := retry(func() error { return os.Rename(tmp, dst) }); err != nil {
		return 0, fmt.Errorf("deploy: %w", err)
	}

	return size, nil
}
