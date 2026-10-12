package install

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"

	"github.com/treejadey/lockdeader/vpk"
)

type VPK struct {
	// Where it is in the archive, with / between directories; "" for a VPK
	// Lockdeader packed from the archive's loose files.
	Path   string
	SHA256 string // in hex; also its name in the store
	Size   int64  // in bytes
}

var (
	ErrNoVPKs        = errors.New("install: the archive has no VPK files in it")
	ErrUnknownFormat = errors.New("install: not a zip, 7z or rar archive, or a VPK")
	// The game only ever loads a mod's _dir.vpk, so a mod with a VPK split
	// into chunks (pak01_000.vpk, …) can't work.
	ErrChunkedVPK = errors.New("install: the archive has a VPK split into several files, which isn't supported")
	ErrTooLarge   = errors.New("install: the archive's files are too big to pack into a VPK")
)

var (
	zipMagic       = []byte("PK\x03\x04")
	emptyZipMagic  = []byte("PK\x05\x06")
	sevenZipMagic  = []byte("7z\xbc\xaf\x27\x1c")
	rarMagic       = []byte("Rar!\x1a\x07")
	vpkMagicBytes  = []byte{0x34, 0x12, 0xaa, 0x55}
	chunkedVPKName = regexp.MustCompile(`(?i)_\d{3}\.vpk$`)
)

// Some mods, mostly sounds, come as the game's compiled files loose in the
// archive instead of in a VPK. Those under one of the game's folders are
// packed into a VPK, as long as they fit in memory comfortably.
var (
	gameFolders  = []string{"sounds", "soundevents", "panorama", "materials", "models", "particles", "scripts", "resource", "maps", "animations"}
	maxLooseSize = int64(256 << 20)
)

// Extract adds every VPK in a zip, 7z or rar archive to the store, in the
// order they're in the archive. The file can also be a VPK itself, in which
// case name, the file's name on GameBanana, is its path. An archive with no
// VPK but with the game's compiled files loose in it gets one VPK packed from
// them.
func Extract(file, name string, store Store) ([]VPK, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	// Tell what kind of file it is from its first few bytes.
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	x := &extraction{store: store}

	switch {
	case bytes.HasPrefix(head, vpkMagicBytes):
		err = x.addVPK(path.Base(name), f)
	case bytes.HasPrefix(head, zipMagic), bytes.HasPrefix(head, emptyZipMagic):
		err = x.extractZip(f, info.Size())
	case bytes.HasPrefix(head, sevenZipMagic):
		err = x.extract7z(f, info.Size())
	case bytes.HasPrefix(head, rarMagic):
		err = x.extractRar(f)
	default:
		return nil, ErrUnknownFormat
	}

	if err != nil {
		return nil, err
	}

	return x.finish()
}

// An extraction collects what an archive has: its VPKs, or failing that its
// loose files to pack into one.
type extraction struct {
	store Store
	vpks  []VPK

	loose     []looseFile
	looseSize int64
}

type looseFile struct {
	path string // in the game's folders
	data []byte
}

func (x *extraction) extractZip(f *os.File, size int64) error {
	z, err := zip.NewReader(f, size)
	if err != nil {
		return fmt.Errorf("install: reading the zip: %w", err)
	}

	for _, entry := range z.File {
		if entry.FileInfo().IsDir() {
			continue
		}

		if err := x.addEntry(entry.Name, entry.Open); err != nil {
			return err
		}
	}

	return nil
}

func (x *extraction) extract7z(f *os.File, size int64) error {
	z, err := sevenzip.NewReader(f, size)
	if err != nil {
		return fmt.Errorf("install: reading the 7z: %w", err)
	}

	for _, entry := range z.File {
		if entry.FileInfo().IsDir() {
			continue
		}

		if err := x.addEntry(entry.Name, entry.Open); err != nil {
			return err
		}
	}

	return nil
}

func (x *extraction) extractRar(f *os.File) error {
	r, err := rardecode.NewReader(f)
	if err != nil {
		return fmt.Errorf("install: reading the rar: %w", err)
	}

	for {
		header, err := r.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("install: reading the rar: %w", err)
		}

		if header.IsDir {
			continue
		}

		// The rar is read in order, so the entry is r itself until Next.
		open := func() (io.ReadCloser, error) { return io.NopCloser(r), nil }

		if err := x.addEntry(header.Name, open); err != nil {
			return err
		}
	}
}

// addEntry takes an archive entry that's a VPK, or a loose file worth packing
// while no VPK has turned up, and leaves the rest, such as readmes.
func (x *extraction) addEntry(name string, open func() (io.ReadCloser, error)) error {
	name = strings.ReplaceAll(name, "\\", "/")

	var loosePath string

	switch {
	case wanted(name):
		// Loose files are only packed when there's no VPK.
		x.loose, x.looseSize = nil, 0
	case len(x.vpks) == 0:
		loosePath = looseGamePath(name)
		if loosePath == "" {
			return nil
		}
	default:
		return nil
	}

	r, err := open()
	if err != nil {
		return fmt.Errorf("install: reading %s from the archive: %w", name, err)
	}

	defer r.Close()

	if loosePath != "" {
		return x.addLoose(loosePath, r)
	}

	return x.addVPK(name, r)
}

// wanted tells the mod's VPKs from everything else, such as a readme, or the
// metadata that macOS adds to archives it makes (__MACOSX/…/._pak01_dir.vpk),
// which isn't a VPK despite its name.
func wanted(name string) bool {
	return strings.EqualFold(path.Ext(name), ".vpk") &&
		!strings.HasPrefix(name, "__MACOSX/") && !strings.HasPrefix(path.Base(name), "._")
}

// looseGamePath returns where a loose compiled file goes in the game, or ""
// for a file that isn't one. The archive may have everything in a folder of
// its own ("My Mod/sounds/…"), which is dropped.
func looseGamePath(name string) string {
	if strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(path.Base(name), "._") {
		return ""
	}

	// Every compiled Source 2 file ends in _c: vsnd_c, vtex_c, vmdl_c, …
	if !strings.HasSuffix(name, "_c") {
		return ""
	}

	parts := strings.Split(name, "/")

	for _, start := range []int{0, 1} {
		if len(parts) > start+1 && slices.Contains(gameFolders, strings.ToLower(parts[start])) {
			return strings.Join(parts[start:], "/")
		}
	}

	return ""
}

func (x *extraction) addLoose(gamePath string, r io.Reader) error {
	// Read one byte past the limit to tell a file that just fits from one
	// that doesn't.
	data, err := io.ReadAll(io.LimitReader(r, maxLooseSize-x.looseSize+1))
	if err != nil {
		return fmt.Errorf("install: reading %s from the archive: %w", gamePath, err)
	}

	x.looseSize += int64(len(data))
	if x.looseSize > maxLooseSize {
		return ErrTooLarge
	}

	x.loose = append(x.loose, looseFile{path: gamePath, data: data})

	return nil
}

func (x *extraction) addVPK(name string, r io.Reader) error {
	if chunkedVPKName.MatchString(name) {
		return fmt.Errorf("%w (%s)", ErrChunkedVPK, name)
	}

	sha, size, err := x.store.Add(r)
	if err != nil {
		return fmt.Errorf("install: %s: %w", name, err)
	}

	if err := x.check(sha); err != nil {
		return fmt.Errorf("install: %s: %w", name, err)
	}

	x.vpks = append(x.vpks, VPK{Path: name, SHA256: sha, Size: size})

	return nil
}

// check makes sure a VPK just added to the store is one the game can load,
// so a broken one never reaches it. One that isn't is removed again: no
// working mod can have the same contents.
func (x *extraction) check(sha string) error {
	f, err := os.Open(x.store.Path(sha))
	if err != nil {
		return err
	}

	d, err := vpk.ReadDir(f)

	// Closed before it may be removed: Windows won't delete an open file.
	f.Close()

	if err != nil {
		x.store.Remove(sha)

		if errors.Is(err, vpk.ErrNotVPK) || errors.Is(err, vpk.ErrVersion) || errors.Is(err, vpk.ErrCorrupt) {
			return fmt.Errorf("%w: %w", ErrNotVPK, err)
		}

		return err
	}

	for _, e := range d.Entries {
		if e.External {
			x.store.Remove(sha)

			return fmt.Errorf("%w (%s)", ErrChunkedVPK, e.Path)
		}
	}

	return nil
}

// finish returns the archive's VPKs, packing its loose files into one if it
// has none.
func (x *extraction) finish() ([]VPK, error) {
	if len(x.vpks) > 0 {
		return x.vpks, nil
	}

	if len(x.loose) == 0 {
		return nil, ErrNoVPKs
	}

	sources := make([]vpk.Source, len(x.loose))

	for i, f := range x.loose {
		sources[i] = vpk.Source{
			Path:  f.path,
			CRC32: crc32.ChecksumIEEE(f.data),
			Size:  uint32(len(f.data)),
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(f.data)), nil
			},
		}
	}

	// Written straight into the store, as it would be read from an archive.
	pr, pw := io.Pipe()

	go func() {
		pw.CloseWithError(vpk.Write(pw, sources))
	}()

	sha, size, err := x.store.Add(pr)
	pr.Close()

	if err != nil {
		return nil, fmt.Errorf("install: packing the archive's files: %w", err)
	}

	return []VPK{{SHA256: sha, Size: size}}, nil
}
