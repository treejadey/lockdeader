// Package rcc writes Qt's binary resource format, as Qt's rcc tool does with
// --binary, so files embedded with go:embed can be handed to Qt
// (QResource::registerResource) and loaded from qrc:/ URLs. The app needs no
// rcc at build time that way. No Qt dependency.
//
// The format is version 1, which every Qt 5 and 6 reads: a header, then a
// tree of nodes, the file names and the file data, uncompressed.
package rcc

import (
	"bytes"
	"encoding/binary"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf16"
)

const (
	flagDirectory = 0x02

	// The locale a file is for: QLocale::C, for any locale. Qt only falls
	// back to a file with no locale of its own when it's C.
	languageC    = 1
	anyTerritory = 0

	nodeSize = 14
)

type node struct {
	name     string
	hash     uint32
	data     []byte
	children []*node // nil for a file
	isDir    bool

	nameOffset int
	dataOffset int
	firstChild int
}

// Build packs every file in fsys, at the same paths: "qml/Main.qml" is
// qrc:/qml/Main.qml once registered at the root.
func Build(fsys fs.FS) ([]byte, error) {
	root := &node{isDir: true}

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		dir := root
		parts := strings.Split(path, "/")

		for _, part := range parts[:len(parts)-1] {
			dir = dir.child(part)
		}

		dir.children = append(dir.children, &node{name: parts[len(parts)-1], hash: hash(parts[len(parts)-1]), data: data})

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Qt finds a directory's children by binary search on their names'
	// hashes, and expects them next to each other, so the nodes are laid
	// out a directory's children at a time, each sorted by hash.
	nodes := []*node{root}

	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		if !n.isDir {
			continue
		}

		slices.SortFunc(n.children, func(a, b *node) int {
			if a.hash != b.hash {
				if a.hash < b.hash {
					return -1
				}

				return 1
			}

			return strings.Compare(a.name, b.name)
		})

		n.firstChild = len(nodes)
		nodes = append(nodes, n.children...)
	}

	var names, payloads bytes.Buffer

	for _, n := range nodes[1:] {
		n.nameOffset = names.Len()
		units := utf16.Encode([]rune(n.name))

		binary.Write(&names, binary.BigEndian, uint16(len(units)))
		binary.Write(&names, binary.BigEndian, n.hash)
		binary.Write(&names, binary.BigEndian, units)

		if !n.isDir {
			n.dataOffset = payloads.Len()

			binary.Write(&payloads, binary.BigEndian, uint32(len(n.data)))
			payloads.Write(n.data)
		}
	}

	var tree bytes.Buffer

	for _, n := range nodes {
		binary.Write(&tree, binary.BigEndian, uint32(n.nameOffset))

		if n.isDir {
			binary.Write(&tree, binary.BigEndian, uint16(flagDirectory))
			binary.Write(&tree, binary.BigEndian, uint32(len(n.children)))
			binary.Write(&tree, binary.BigEndian, uint32(n.firstChild))
		} else {
			binary.Write(&tree, binary.BigEndian, uint16(0))
			binary.Write(&tree, binary.BigEndian, uint16(anyTerritory))
			binary.Write(&tree, binary.BigEndian, uint16(languageC))
			binary.Write(&tree, binary.BigEndian, uint32(n.dataOffset))
		}
	}

	// The header: magic, version, then where the tree, the data and the
	// names start.
	const headerSize = 20

	var out bytes.Buffer
	out.WriteString("qres")
	binary.Write(&out, binary.BigEndian, uint32(1))
	binary.Write(&out, binary.BigEndian, uint32(headerSize))
	binary.Write(&out, binary.BigEndian, uint32(headerSize+tree.Len()))
	binary.Write(&out, binary.BigEndian, uint32(headerSize+tree.Len()+payloads.Len()))
	out.Write(tree.Bytes())
	out.Write(payloads.Bytes())
	out.Write(names.Bytes())

	return out.Bytes(), nil
}

func (n *node) child(name string) *node {
	for _, c := range n.children {
		if c.isDir && c.name == name {
			return c
		}
	}

	c := &node{name: name, hash: hash(name), isDir: true}
	n.children = append(n.children, c)

	return c
}

// hash is Qt's qt_hash, over the name's UTF-16.
func hash(name string) uint32 {
	var h uint32

	for _, u := range utf16.Encode([]rune(name)) {
		h = (h << 4) + uint32(u)
		h ^= (h & 0xf0000000) >> 23
		h &= 0x0fffffff
	}

	return h
}
