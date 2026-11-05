// Package resource reads the container Source 2 compiles its files into
// (the _c files: .vsndevts_c, .vdata_c and so on): a header and a list of
// named blocks, such as DATA, which holds what the file is, and RED2, which
// records what it was compiled from. What a block holds is left to other
// packages, e.g. kv3 for a DATA block in binary KV3.
package resource

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// The only header version Source 2 has written.
const headerVersion = 12

var (
	ErrNotResource = errors.New("resource: not a compiled Source 2 file")
	ErrCorrupt     = errors.New("resource: the file is damaged")
)

// A Resource is a compiled file's blocks.
type Resource struct {
	// The version of the resource type, e.g. 1 for sound events.
	Version uint16
	Blocks  []Block
}

// A Block is one of a resource's blocks.
type Block struct {
	Type string // four characters, e.g. "DATA"
	Data []byte // part of the data given to Read
}

// Read reads the blocks of the resource in data. The blocks' data is not
// copied.
func Read(data []byte) (*Resource, error) {
	if len(data) < 16 {
		return nil, ErrNotResource
	}

	le := binary.LittleEndian

	// data[0:4] is the file's size, which nothing needs and some tools get
	// wrong.
	if le.Uint16(data[4:]) != headerVersion {
		return nil, ErrNotResource
	}

	r := &Resource{Version: le.Uint16(data[6:])}

	// The offset of the block list is counted from where it's stored, as
	// every offset in the format is.
	start := 8 + uint64(le.Uint32(data[8:]))
	count := uint64(le.Uint32(data[12:]))

	if start+count*12 > uint64(len(data)) {
		return nil, fmt.Errorf("%w: its list of blocks is cut off", ErrCorrupt)
	}

	for i := range count {
		at := start + i*12
		offset := at + 4 + uint64(le.Uint32(data[at+4:]))
		size := uint64(le.Uint32(data[at+8:]))

		if offset+size > uint64(len(data)) {
			return nil, fmt.Errorf("%w: block %d is outside the file", ErrCorrupt, i)
		}

		r.Blocks = append(r.Blocks, Block{
			Type: string(data[at : at+4]),
			Data: data[offset : offset+size : offset+size],
		})
	}

	return r, nil
}

// Block returns the data of the first block of the type given.
func (r *Resource) Block(typ string) ([]byte, bool) {
	for _, b := range r.Blocks {
		if b.Type == typ {
			return b.Data, true
		}
	}

	return nil, false
}

// Write lays out a resource as Source 2's resource compiler does: the
// header, the block list, then each block's data, starting on a 16-byte
// boundary.
func Write(r *Resource) []byte {
	le := binary.LittleEndian

	var out []byte

	out = le.AppendUint32(out, 0) // the file's size, filled in at the end
	out = le.AppendUint16(out, headerVersion)
	out = le.AppendUint16(out, r.Version)
	out = le.AppendUint32(out, 8) // the block list follows, 8 bytes on
	out = le.AppendUint32(out, uint32(len(r.Blocks)))

	list := len(out)
	out = append(out, make([]byte, 12*len(r.Blocks))...)

	for i, b := range r.Blocks {
		out = pad(out, 16)

		at := list + 12*i
		copy(out[at:], b.Type)
		le.PutUint32(out[at+4:], uint32(len(out)-(at+4)))
		le.PutUint32(out[at+8:], uint32(len(b.Data)))

		out = append(out, b.Data...)
	}

	le.PutUint32(out, uint32(len(out)))

	return out
}

func pad(b []byte, n int) []byte {
	for len(b)%n != 0 {
		b = append(b, 0)
	}

	return b
}
