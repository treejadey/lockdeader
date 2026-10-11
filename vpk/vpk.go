// Package vpk reads and writes version 2 VPK files, the archives Source 2
// games load their files from. It reads only the directory tree, never the
// file data, which is all that telling which files a mod has takes; and it
// writes new VPKs by streaming the data of files given to it, so mods can be
// packed and merged without holding them in memory. See
// docs/deadlock-modding.md.
package vpk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	magic      = 0x55AA1234
	version    = 2
	headerSize = 28
	// The archive index of a file whose data is in the _dir.vpk itself, after
	// the tree, rather than in a pakNN_NNN.vpk chunk.
	embedded = 0x7FFF
	// After each file's entry.
	terminator = 0xFFFF
	entrySize  = 18
	// The sizes of the checksums that follow the file data: the tree's MD5,
	// the archive MD5 section's MD5, and the whole file's.
	checksumsSize = 48
	// Mods' trees are a few KB; the base game's is a few MB.
	maxTreeSize = 64 << 20
)

var (
	ErrNotVPK  = errors.New("vpk: not a VPK file")
	ErrVersion = errors.New("vpk: not a version 2 VPK")
	ErrCorrupt = errors.New("vpk: the file's directory is damaged")
	// ErrDuplicate is returned by Write for two sources with the same path.
	ErrDuplicate = errors.New("vpk: the same path is given twice")
	// ErrTooBig is returned by Write when the data wouldn't fit the format's
	// 32-bit offsets.
	ErrTooBig = errors.New("vpk: the files are too big for one VPK")
)

// An Entry is a file in a VPK.
type Entry struct {
	Path  string // dir/name.ext, with / between directories, as written in the VPK
	CRC32 uint32 // of the file's data
	Size  uint32 // of the file's data, in bytes
	// Its data is in a pakNN_NNN.vpk chunk, not in this file. Only the base
	// game is split that way; a mod never is.
	External bool

	// The start of the data is kept in the tree, the rest after it.
	preload        []byte
	offset, length uint32
}

// A Dir is a VPK's directory: its files, and where their data is.
type Dir struct {
	Entries []Entry

	r         io.ReaderAt
	dataStart int64
	dataSize  uint32
}

// ReadDir reads a VPK's directory from r, which must start at the VPK's
// beginning. It reads only the header and the tree.
func ReadDir(r io.ReaderAt) (*Dir, error) {
	var header [headerSize]byte

	if _, err := io.ReadFull(io.NewSectionReader(r, 0, headerSize), header[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrNotVPK
		}

		return nil, fmt.Errorf("vpk: %w", err)
	}

	fields := make([]uint32, 7)

	for i := range fields {
		fields[i] = binary.LittleEndian.Uint32(header[i*4:])
	}

	if fields[0] != magic {
		return nil, ErrNotVPK
	}

	if fields[1] != version {
		return nil, ErrVersion
	}

	treeSize, dataSize := fields[2], fields[3]
	if treeSize > maxTreeSize {
		return nil, ErrCorrupt
	}

	tree := make([]byte, treeSize)

	if _, err := io.ReadFull(io.NewSectionReader(r, headerSize, int64(treeSize)), tree); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrCorrupt
		}

		return nil, fmt.Errorf("vpk: %w", err)
	}

	d := &Dir{r: r, dataStart: headerSize + int64(treeSize), dataSize: dataSize}

	// A VPK with no files has no tree at all.
	if treeSize == 0 {
		return d, nil
	}

	if err := d.parseTree(tree); err != nil {
		return nil, err
	}

	return d, nil
}

// The tree nests null-terminated strings: extension, directory, file name,
// each level ending with an empty string. A space stands for no extension,
// and for the root directory.
func (d *Dir) parseTree(tree []byte) error {
	p := &treeParser{data: tree}

	for {
		ext, err := p.readString()
		if err != nil {
			return err
		}

		if ext == "" {
			return nil
		}

		for {
			dir, err := p.readString()
			if err != nil {
				return err
			}

			if dir == "" {
				break
			}

			for {
				name, err := p.readString()
				if err != nil {
					return err
				}

				if name == "" {
					break
				}

				e, err := p.readEntry()
				if err != nil {
					return err
				}

				if !e.External && uint64(e.offset)+uint64(e.length) > uint64(d.dataSize) {
					return ErrCorrupt
				}

				e.Path = joinPath(dir, name, ext)
				d.Entries = append(d.Entries, e)
			}
		}
	}
}

type treeParser struct {
	data []byte
	pos  int
}

func (p *treeParser) readString() (string, error) {
	end := bytes.IndexByte(p.data[p.pos:], 0)
	if end < 0 {
		return "", ErrCorrupt
	}

	s := string(p.data[p.pos : p.pos+end])
	p.pos += end + 1

	return s, nil
}

func (p *treeParser) readEntry() (Entry, error) {
	if len(p.data)-p.pos < entrySize {
		return Entry{}, ErrCorrupt
	}

	b := p.data[p.pos:]
	preloadBytes := binary.LittleEndian.Uint16(b[4:])
	archive := binary.LittleEndian.Uint16(b[6:])

	e := Entry{
		CRC32:    binary.LittleEndian.Uint32(b),
		External: archive != embedded,
		offset:   binary.LittleEndian.Uint32(b[8:]),
		length:   binary.LittleEndian.Uint32(b[12:]),
	}

	if binary.LittleEndian.Uint16(b[16:]) != terminator {
		return Entry{}, ErrCorrupt
	}

	p.pos += entrySize

	if len(p.data)-p.pos < int(preloadBytes) {
		return Entry{}, ErrCorrupt
	}

	e.preload = p.data[p.pos : p.pos+int(preloadBytes)]
	p.pos += int(preloadBytes)
	e.Size = uint32(preloadBytes) + e.length

	return e, nil
}

func joinPath(dir, name, ext string) string {
	path := name

	if ext != " " {
		path += "." + ext
	}

	if dir != " " {
		path = dir + "/" + path
	}

	return path
}

// splitPath is the reverse of joinPath.
func splitPath(path string) (dir, name, ext string) {
	dir, name = " ", path

	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		dir, name = path[:i], path[i+1:]
	}

	ext = " "

	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name, ext = name[:i], name[i+1:]
	}

	return dir, name, ext
}

// Open returns the data of one of d's files. For an External entry it's only
// the part kept in the tree, since the rest is in another file.
func (d *Dir) Open(e Entry) io.Reader {
	if e.External {
		return bytes.NewReader(e.preload)
	}

	return io.MultiReader(bytes.NewReader(e.preload),
		io.NewSectionReader(d.r, d.dataStart+int64(e.offset), int64(e.length)))
}
