package rcc

import (
	"bytes"
	"encoding/binary"
	"testing"
	"testing/fstest"
	"unicode/utf16"
)

// The hashes Qt's rcc (6.11) wrote for these names.
func TestHash(t *testing.T) {
	for name, want := range map[string]uint32{
		"qml":                           0x0000783c,
		"icons":                         0x006fa653,
		"Main.qml":                      0x08015e5c,
		"DownloadProgress.qml":          0x05c41d7c,
		"bitmap":                        0x0690b380,
		"arrow_download_24_regular.svg": 0x0759aa67,
		"sound_icon.png":                0x07e80227,
	} {
		if got := hash(name); got != want {
			t.Errorf("hash(%q) = %#08x, want %#08x", name, got, want)
		}
	}
}

// lookup finds a file the way Qt's QResourceRoot::findNode does: down the
// tree, by binary search on each directory's children's hashes.
func lookup(t *testing.T, data []byte, path string) ([]byte, bool) {
	t.Helper()

	be := binary.BigEndian

	if string(data[:4]) != "qres" || be.Uint32(data[4:]) != 1 {
		t.Fatalf("bad header % x", data[:8])
	}

	tree, payloads, names := data[be.Uint32(data[8:]):], data[be.Uint32(data[12:]):], data[be.Uint32(data[16:]):]

	nodeHash := func(n uint32) uint32 { return be.Uint32(names[be.Uint32(tree[n*nodeSize:])+2:]) }

	nodeName := func(n uint32) string {
		offset := be.Uint32(tree[n*nodeSize:])
		units := make([]uint16, be.Uint16(names[offset:]))
		binary.Read(bytes.NewReader(names[offset+6:]), be, units)

		return string(utf16.Decode(units))
	}

	var n uint32

	for _, part := range bytes.Split([]byte(path), []byte("/")) {
		node := tree[n*nodeSize:]
		if be.Uint16(node[4:])&flagDirectory == 0 {
			return nil, false
		}

		count, first := be.Uint32(node[6:]), be.Uint32(node[10:])
		h := hash(string(part))

		lo, hi := 0, int(count)-1

		for lo <= hi {
			mid := (lo + hi) / 2
			if nodeHash(first+uint32(mid)) < h {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}

		found := false

		for i := uint32(lo); i < count && nodeHash(first+i) == h; i++ {
			if nodeName(first+i) == string(part) {
				n, found = first+i, true

				break
			}
		}

		if !found {
			return nil, false
		}
	}

	node := tree[n*nodeSize:]
	if be.Uint16(node[4:])&flagDirectory != 0 || be.Uint16(node[8:]) != languageC {
		return nil, false
	}

	offset := be.Uint32(node[10:])

	return payloads[offset+4 : offset+4+be.Uint32(payloads[offset:])], true
}

func TestBuild(t *testing.T) {
	files := fstest.MapFS{
		"qml/Main.qml":             {Data: []byte("import QtQuick")},
		"qml/DownloadProgress.qml": {Data: []byte("Item {}")},
		"qml/qmldir":               {Data: []byte("singleton Theme 1.0 Theme.qml")},
		"qml/icons/arrow_download_24_regular.svg":   {Data: []byte("<svg/>")},
		"qml/icons/bitmap/sound_icon.png":           {Data: []byte{0x89, 'P', 'N', 'G'}},
		"qml/Ünïcödé.qml":                           {Data: []byte("")},
		"other/a_very_long_name_past_the_hash_mask": {Data: []byte("x")},
	}

	data, err := Build(files)
	if err != nil {
		t.Fatal(err)
	}

	for path, file := range files {
		got, ok := lookup(t, data, path)
		if !ok || !bytes.Equal(got, file.Data) {
			t.Errorf("%s: found %v, %q; want %q", path, ok, got, file.Data)
		}
	}

	for _, path := range []string{"qml/Missing.qml", "qml/icons/bitmap", "nope/Main.qml"} {
		if _, ok := lookup(t, data, path); ok {
			t.Errorf("%s was found", path)
		}
	}
}
