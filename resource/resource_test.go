package resource

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

type testBlock struct {
	typ  string
	data string
}

// build lays out a resource the way resourcecompiler does: the header, the
// block list right after it, then each block's data.
func build(blocks ...testBlock) []byte {
	le := binary.LittleEndian

	var b []byte

	b = le.AppendUint32(b, 0)
	b = le.AppendUint16(b, headerVersion)
	b = le.AppendUint16(b, 1)
	b = le.AppendUint32(b, 8)
	b = le.AppendUint32(b, uint32(len(blocks)))

	dataStart := len(b) + 12*len(blocks)
	next := dataStart

	for _, blk := range blocks {
		at := len(b)
		b = append(b, blk.typ...)
		b = le.AppendUint32(b, uint32(next-(at+4)))
		b = le.AppendUint32(b, uint32(len(blk.data)))
		next += len(blk.data)
	}

	for _, blk := range blocks {
		b = append(b, blk.data...)
	}

	le.PutUint32(b, uint32(len(b)))

	return b
}

func TestRead(t *testing.T) {
	data := build(testBlock{"RED2", "edit info"}, testBlock{"DATA", "the data"})

	r, err := Read(data)
	if err != nil {
		t.Fatal(err)
	}

	if r.Version != 1 || len(r.Blocks) != 2 {
		t.Fatalf("Read = version %d, %d blocks, want version 1, 2 blocks", r.Version, len(r.Blocks))
	}

	got, ok := r.Block("DATA")
	if !ok || !bytes.Equal(got, []byte("the data")) {
		t.Errorf("Block(DATA) = %q, %v", got, ok)
	}

	if r.Blocks[0].Type != "RED2" || string(r.Blocks[0].Data) != "edit info" {
		t.Errorf("first block = %q %q", r.Blocks[0].Type, r.Blocks[0].Data)
	}

	if _, ok := r.Block("NTRO"); ok {
		t.Error("Block(NTRO) found a block the resource doesn't have")
	}
}

func TestReadNotResource(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("short"), []byte("a VPK or anything else at all")} {
		if _, err := Read(data); !errors.Is(err, ErrNotResource) {
			t.Errorf("Read(%q) = %v, want ErrNotResource", data, err)
		}
	}
}

func TestReadDamaged(t *testing.T) {
	data := build(testBlock{"DATA", "the data"})

	// Cut off inside the block's data, then inside the block list.
	for _, n := range []int{len(data) - 1, 20} {
		if _, err := Read(data[:n]); !errors.Is(err, ErrCorrupt) {
			t.Errorf("Read of %d of %d bytes = %v, want ErrCorrupt", n, len(data), err)
		}
	}

	// A block offset that wraps around.
	bad := bytes.Clone(data)
	binary.LittleEndian.PutUint32(bad[20:], 0xFFFFFFF0)

	if _, err := Read(bad); !errors.Is(err, ErrCorrupt) {
		t.Errorf("Read with a block outside the file = %v, want ErrCorrupt", err)
	}
}

func TestWrite(t *testing.T) {
	want := &Resource{Version: 1, Blocks: []Block{{"RED2", []byte("edit info")}, {"DATA", []byte("the data")}}}
	data := Write(want)

	got, err := Read(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != want.Version || len(got.Blocks) != 2 {
		t.Fatalf("Read(Write) = version %d, %d blocks", got.Version, len(got.Blocks))
	}

	for i, b := range got.Blocks {
		if b.Type != want.Blocks[i].Type || !bytes.Equal(b.Data, want.Blocks[i].Data) {
			t.Errorf("block %d = %q %q, want %q %q", i, b.Type, b.Data, want.Blocks[i].Type, want.Blocks[i].Data)
		}

		// Each block starts on a 16-byte boundary, as the resource compiler
		// writes them.
		if at := bytes.Index(data, b.Data); at%16 != 0 {
			t.Errorf("block %d starts at %d", i, at)
		}
	}

	if size := binary.LittleEndian.Uint32(data); int(size) != len(data) {
		t.Errorf("size field %d, want %d", size, len(data))
	}
}
