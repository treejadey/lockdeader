package vpk

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"slices"
	"strings"
	"testing"
)

func source(path, data string) Source {
	return Source{
		Path:  path,
		CRC32: crc32.ChecksumIEEE([]byte(data)),
		Size:  uint32(len(data)),
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(data)), nil
		},
	}
}

func write(t *testing.T, sources ...Source) []byte {
	t.Helper()

	var b bytes.Buffer

	if err := Write(&b, sources); err != nil {
		t.Fatal(err)
	}

	return b.Bytes()
}

func read(t *testing.T, data []byte) *Dir {
	t.Helper()

	d, err := ReadDir(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func contents(t *testing.T, d *Dir, e Entry) string {
	t.Helper()

	data, err := io.ReadAll(d.Open(e))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestRoundTrip(t *testing.T) {
	files := map[string]string{
		"sounds/music/intro.vsnd_c":        "intro music",
		"soundevents/music.vsndevts_c":     "events",
		"panorama/layout/chat.vxml_c":      "<root/>",
		"panorama/scripts/chat.vjs_c":      "js",
		"readme":                           "no extension",
		"root.txt":                         "in the root",
		"models/heroes/a/b/c/model.vmdl_c": "",
	}

	var sources []Source

	for path, data := range files {
		sources = append(sources, source(path, data))
	}

	data := write(t, sources...)
	d := read(t, data)

	if len(d.Entries) != len(files) {
		t.Fatalf("got %d entries, want %d", len(d.Entries), len(files))
	}

	for _, e := range d.Entries {
		want, ok := files[e.Path]
		if !ok {
			t.Errorf("unexpected entry %q", e.Path)

			continue
		}

		if e.External {
			t.Errorf("%s: external", e.Path)
		}

		if e.Size != uint32(len(want)) {
			t.Errorf("%s: size %d, want %d", e.Path, e.Size, len(want))
		}

		if e.CRC32 != crc32.ChecksumIEEE([]byte(want)) {
			t.Errorf("%s: wrong CRC", e.Path)
		}

		if got := contents(t, d, e); got != want {
			t.Errorf("%s: contents %q, want %q", e.Path, got, want)
		}
	}

	// The same sources in another order give the same bytes.
	slices.Reverse(sources)

	if !bytes.Equal(write(t, sources...), data) {
		t.Error("the output depends on the order of the sources")
	}
}

func TestWriteChecksums(t *testing.T) {
	data := write(t, source("a/b.c", "hello"), source("d.e", "world"))

	treeSize := binary.LittleEndian.Uint32(data[8:])
	dataSize := binary.LittleEndian.Uint32(data[12:])

	if dataSize != 10 {
		t.Errorf("data size %d, want 10", dataSize)
	}

	if binary.LittleEndian.Uint32(data[20:]) != checksumsSize {
		t.Error("the header doesn't say the checksums are there")
	}

	tree := data[headerSize : headerSize+treeSize]
	end := headerSize + int(treeSize) + int(dataSize)

	if len(data) != end+checksumsSize {
		t.Fatalf("file is %d bytes, want %d", len(data), end+checksumsSize)
	}

	sums := data[end:]
	treeSum, archiveSum, wholeSum := md5.Sum(tree), md5.Sum(nil), md5.Sum(data[:end+32])

	if !bytes.Equal(sums[:16], treeSum[:]) || !bytes.Equal(sums[16:32], archiveSum[:]) || !bytes.Equal(sums[32:], wholeSum[:]) {
		t.Error("wrong checksums")
	}
}

func TestWriteEmpty(t *testing.T) {
	d := read(t, write(t))

	if len(d.Entries) != 0 {
		t.Errorf("got %d entries", len(d.Entries))
	}
}

func TestWriteDuplicate(t *testing.T) {
	err := Write(io.Discard, []Source{source("a/b.c", "1"), source("A/B.C", "2")})
	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("got %v, want ErrDuplicate", err)
	}
}

func TestWriteShortSource(t *testing.T) {
	s := source("a.b", "hello")
	s.Size = 10

	if err := Write(io.Discard, []Source{s}); err == nil {
		t.Error("a source shorter than its size was accepted")
	}
}

// handTree builds a tree by hand, with a preload and an external entry,
// which Write never makes.
func handTree(t *testing.T) []byte {
	t.Helper()

	var tree []byte
	add := func(s string) { tree = append(tree, s...); tree = append(tree, 0) }
	entry := func(crc uint32, preload string, archive uint16, offset, length uint32) {
		tree = binary.LittleEndian.AppendUint32(tree, crc)
		tree = binary.LittleEndian.AppendUint16(tree, uint16(len(preload)))
		tree = binary.LittleEndian.AppendUint16(tree, archive)
		tree = binary.LittleEndian.AppendUint32(tree, offset)
		tree = binary.LittleEndian.AppendUint32(tree, length)
		tree = binary.LittleEndian.AppendUint16(tree, terminator)
		tree = append(tree, preload...)
	}

	add("txt")
	add(" ") // the root
	add("hello")
	entry(1, "hel", embedded, 0, 2)
	add("")
	add("dir")
	add("chunked")
	entry(2, "", 3, 100, 50)
	add("")
	add("")
	add(" ") // no extension
	add("dir")
	add("plain")
	entry(3, "", embedded, 2, 3)
	add("")
	add("")
	add("")

	return tree
}

func file(tree []byte, data string) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [7]uint32{magic, version, uint32(len(tree)), uint32(len(data))})
	b.Write(tree)
	b.WriteString(data)

	return b.Bytes()
}

func TestReadDirHandBuilt(t *testing.T) {
	d := read(t, file(handTree(t), "lo!!!"))

	want := []Entry{
		{Path: "hello.txt", CRC32: 1, Size: 5},
		{Path: "dir/chunked.txt", CRC32: 2, Size: 50, External: true},
		{Path: "dir/plain", CRC32: 3, Size: 3},
	}

	if len(d.Entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(d.Entries), len(want))
	}

	for i, e := range d.Entries {
		w := want[i]

		if e.Path != w.Path || e.CRC32 != w.CRC32 || e.Size != w.Size || e.External != w.External {
			t.Errorf("entry %d: %+v, want %+v", i, e, w)
		}
	}

	if got := contents(t, d, d.Entries[0]); got != "hello" {
		t.Errorf("preload entry reads %q", got)
	}

	if got := contents(t, d, d.Entries[2]); got != "!!!" {
		t.Errorf("embedded entry reads %q", got)
	}

	if got := contents(t, d, d.Entries[1]); got != "" {
		t.Errorf("external entry reads %q", got)
	}
}

func TestReadDirErrors(t *testing.T) {
	tree := handTree(t)
	good := file(tree, "lo!!!")

	tests := map[string]struct {
		data []byte
		err  error
	}{
		"empty":            {nil, ErrNotVPK},
		"short header":     {good[:10], ErrNotVPK},
		"wrong magic":      {append([]byte("nope"), good[4:]...), ErrNotVPK},
		"version 1":        {file(tree, "lo!!!")[:4:4], ErrNotVPK},
		"truncated tree":   {good[:headerSize+len(tree)-4], ErrCorrupt},
		"no terminator":    {func() []byte { b := slices.Clone(good); b[headerSize+len("txt\x00 \x00hello\x00")+16] = 0; return b }(), ErrCorrupt},
		"data past end":    {func() []byte { b := slices.Clone(good); binary.LittleEndian.PutUint32(b[12:], 2); return b }(), ErrCorrupt},
		"huge tree":        {func() []byte { b := slices.Clone(good); binary.LittleEndian.PutUint32(b[8:], maxTreeSize+1); return b }(), ErrCorrupt},
		"header only":      {good[:headerSize], ErrCorrupt},
		"only a few bytes": {good[:headerSize+3], ErrCorrupt},
	}

	// A version 1 file has the magic but a different version.
	v1 := slices.Clone(good)
	binary.LittleEndian.PutUint32(v1[4:], 1)
	tests["version 1"] = struct {
		data []byte
		err  error
	}{v1, ErrVersion}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ReadDir(bytes.NewReader(tc.data))
			if !errors.Is(err, tc.err) {
				t.Errorf("got %v, want %v", err, tc.err)
			}
		})
	}
}

func TestReadDirNoTree(t *testing.T) {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [7]uint32{magic, version})
	b.WriteString("trailing bytes the game ignores")

	d := read(t, b.Bytes())

	if len(d.Entries) != 0 {
		t.Errorf("got %d entries", len(d.Entries))
	}
}
