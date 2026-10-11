package install

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

// VPK is a VPK taken out of a mod's archive and added to the store.
type VPK struct {
	Path   string // where it is in the archive, with / between directories
	SHA256 string // in hex; also its name in the store
	Size   int64  // in bytes
}

var (
	// ErrNoVPKs is returned by Extract for an archive with no VPKs in it.
	ErrNoVPKs = errors.New("install: the archive has no VPK files in it")
	// ErrUnknownFormat is returned by Extract for a file that's neither a
	// VPK nor an archive it can open.
	ErrUnknownFormat = errors.New("install: not a zip, 7z or rar archive, or a VPK")
	// ErrChunkedVPK is returned by Extract for an archive with a VPK split
	// into chunks (pak01_000.vpk, …). The game only ever loads a mod's
	// _dir.vpk, so such a mod can't work.
	ErrChunkedVPK = errors.New("install: the archive has a VPK split into several files, which isn't supported")
)

// The first bytes of each kind of file Extract opens.
var (
	zipMagic       = []byte("PK\x03\x04")
	emptyZipMagic  = []byte("PK\x05\x06")
	sevenZipMagic  = []byte("7z\xbc\xaf\x27\x1c")
	rarMagic       = []byte("Rar!\x1a\x07")
	vpkMagicBytes  = []byte{0x34, 0x12, 0xaa, 0x55}
	chunkedVPKName = regexp.MustCompile(`(?i)_\d{3}\.vpk$`)
)

// Extract adds every VPK in the archive at file to the store, and returns
// them in the order they're in the archive. The archive is a zip, 7z or rar,
// or the file can be a VPK itself, in which case name, the file's name on
// GameBanana, is its path.
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

	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	var vpks []VPK
	switch {
	case bytes.HasPrefix(head, vpkMagicBytes):
		vpks, err = addVPK(store, nil, path.Base(name), f)
	case bytes.HasPrefix(head, zipMagic), bytes.HasPrefix(head, emptyZipMagic):
		vpks, err = extractZip(f, info.Size(), store)
	case bytes.HasPrefix(head, sevenZipMagic):
		vpks, err = extract7z(f, info.Size(), store)
	case bytes.HasPrefix(head, rarMagic):
		vpks, err = extractRar(f, store)
	default:
		return nil, ErrUnknownFormat
	}
	if err != nil {
		return nil, err
	}
	if len(vpks) == 0 {
		return nil, ErrNoVPKs
	}
	return vpks, nil
}

func extractZip(f *os.File, size int64, store Store) ([]VPK, error) {
	z, err := zip.NewReader(f, size)
	if err != nil {
		return nil, fmt.Errorf("install: reading the zip: %w", err)
	}
	var vpks []VPK
	for _, entry := range z.File {
		if entry.FileInfo().IsDir() || !wanted(entry.Name) {
			continue
		}
		if vpks, err = addEntry(store, vpks, entry.Name, entry.Open); err != nil {
			return nil, err
		}
	}
	return vpks, nil
}

func extract7z(f *os.File, size int64, store Store) ([]VPK, error) {
	z, err := sevenzip.NewReader(f, size)
	if err != nil {
		return nil, fmt.Errorf("install: reading the 7z: %w", err)
	}
	var vpks []VPK
	for _, entry := range z.File {
		if entry.FileInfo().IsDir() || !wanted(entry.Name) {
			continue
		}
		if vpks, err = addEntry(store, vpks, entry.Name, entry.Open); err != nil {
			return nil, err
		}
	}
	return vpks, nil
}

func extractRar(f *os.File, store Store) ([]VPK, error) {
	r, err := rardecode.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("install: reading the rar: %w", err)
	}
	var vpks []VPK
	for {
		header, err := r.Next()
		if errors.Is(err, io.EOF) {
			return vpks, nil
		}
		if err != nil {
			return nil, fmt.Errorf("install: reading the rar: %w", err)
		}
		if header.IsDir || !wanted(header.Name) {
			continue
		}
		if vpks, err = addVPK(store, vpks, header.Name, r); err != nil {
			return nil, err
		}
	}
}

// wanted reports whether the file at name in an archive is one of the mod's
// VPKs, rather than e.g. a readme, or the metadata that macOS adds to
// archives it makes (__MACOSX/…/._pak01_dir.vpk), which isn't a VPK at all.
func wanted(name string) bool {
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.EqualFold(path.Ext(name), ".vpk") &&
		!strings.HasPrefix(name, "__MACOSX/") && !strings.HasPrefix(path.Base(name), "._")
}

func addEntry(store Store, vpks []VPK, name string, open func() (io.ReadCloser, error)) ([]VPK, error) {
	r, err := open()
	if err != nil {
		return nil, fmt.Errorf("install: reading %s from the archive: %w", name, err)
	}
	defer r.Close()
	return addVPK(store, vpks, name, r)
}

// addVPK adds the VPK at name in an archive to the store, and appends it to
// vpks.
func addVPK(store Store, vpks []VPK, name string, r io.Reader) ([]VPK, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if chunkedVPKName.MatchString(name) {
		return nil, fmt.Errorf("%w (%s)", ErrChunkedVPK, name)
	}
	sha, size, err := store.Add(r)
	if err != nil {
		return nil, fmt.Errorf("install: %s: %w", name, err)
	}
	return append(vpks, VPK{Path: name, SHA256: sha, Size: size}), nil
}
