package vpk

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"slices"
	"strings"
)

// A Source is a file to write into a VPK. Its size and CRC-32 have to be
// known up front, because the tree comes before the data; from a VPK they're
// in its Entry.
type Source struct {
	Path  string // dir/name.ext, with / between directories
	CRC32 uint32
	Size  uint32
	Open  func() (io.ReadCloser, error)
}

// Write writes one VPK holding sources, with all of their data in it, in
// the order that produces the same bytes for the same sources. Each source
// is opened and copied in turn, so the data is never all in memory.
func Write(w io.Writer, sources []Source) error {
	sources = slices.Clone(sources)

	slices.SortStableFunc(sources, func(a, b Source) int {
		ad, an, ae := splitPath(a.Path)
		bd, bn, be := splitPath(b.Path)

		return strings.Compare(ae+"\x00"+ad+"\x00"+an, be+"\x00"+bd+"\x00"+bn)
	})

	// The game treats paths case-insensitively, so so does this.
	seen := make(map[string]bool, len(sources))

	for _, s := range sources {
		key := strings.ToLower(s.Path)
		if seen[key] {
			return fmt.Errorf("%w: %s", ErrDuplicate, s.Path)
		}

		seen[key] = true
	}

	tree, err := buildTree(sources)
	if err != nil {
		return err
	}

	var dataSize uint64

	for _, s := range sources {
		dataSize += uint64(s.Size)
	}

	header := make([]byte, headerSize)

	for i, v := range []uint32{magic, version, uint32(len(tree)), uint32(dataSize), 0, checksumsSize, 0} {
		binary.LittleEndian.PutUint32(header[i*4:], v)
	}

	// Everything before the whole-file checksum is part of it.
	whole := md5.New()
	out := io.MultiWriter(w, whole)

	if _, err := out.Write(header); err != nil {
		return fmt.Errorf("vpk: %w", err)
	}

	if _, err := out.Write(tree); err != nil {
		return fmt.Errorf("vpk: %w", err)
	}

	for _, s := range sources {
		if err := copySource(out, s); err != nil {
			return err
		}
	}

	if err := writeChecksums(w, out, whole, tree); err != nil {
		return fmt.Errorf("vpk: %w", err)
	}

	return nil
}

// buildTree lays sources out by extension, then directory, then name, as
// Write has sorted them, with each file's data following the one before.
func buildTree(sources []Source) ([]byte, error) {
	var tree []byte
	var offset uint64
	lastExt, lastDir := "", ""

	for _, s := range sources {
		dir, name, ext := splitPath(s.Path)

		if ext != lastExt || lastExt == "" {
			if lastExt != "" {
				tree = append(tree, 0, 0) // end of the directory, and of the extension
			}

			tree = append(tree, ext...)
			tree = append(tree, 0)
			lastExt, lastDir = ext, ""
		}

		if dir != lastDir || lastDir == "" {
			if lastDir != "" {
				tree = append(tree, 0) // end of the directory
			}

			tree = append(tree, dir...)
			tree = append(tree, 0)
			lastDir = dir
		}

		if offset+uint64(s.Size) > 0xFFFFFFFF {
			return nil, ErrTooBig
		}

		tree = append(tree, name...)
		tree = append(tree, 0)
		tree = binary.LittleEndian.AppendUint32(tree, s.CRC32)
		tree = binary.LittleEndian.AppendUint16(tree, 0) // no preload bytes
		tree = binary.LittleEndian.AppendUint16(tree, embedded)
		tree = binary.LittleEndian.AppendUint32(tree, uint32(offset))
		tree = binary.LittleEndian.AppendUint32(tree, s.Size)
		tree = binary.LittleEndian.AppendUint16(tree, terminator)

		offset += uint64(s.Size)
	}

	if lastExt != "" {
		tree = append(tree, 0, 0)
	}

	// The end of the tree.
	tree = append(tree, 0)

	return tree, nil
}

func copySource(w io.Writer, s Source) (err error) {
	r, err := s.Open()
	if err != nil {
		return fmt.Errorf("vpk: %s: %w", s.Path, err)
	}

	defer func() {
		if closeErr := r.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("vpk: %s: %w", s.Path, closeErr)
		}
	}()

	n, err := io.CopyN(w, r, int64(s.Size))
	if err != nil {
		return fmt.Errorf("vpk: %s: %w", s.Path, err)
	}

	if n != int64(s.Size) {
		return fmt.Errorf("vpk: %s: shorter than its size", s.Path)
	}

	return nil
}

// writeChecksums writes the three MD5s the format ends with. The whole-file
// one covers the two before it, so they go through out, which feeds whole.
func writeChecksums(w, out io.Writer, whole hash.Hash, tree []byte) error {
	treeSum := md5.Sum(tree)
	archiveSum := md5.Sum(nil) // there's no archive MD5 section

	if _, err := out.Write(treeSum[:]); err != nil {
		return err
	}

	if _, err := out.Write(archiveSum[:]); err != nil {
		return err
	}

	_, err := w.Write(whole.Sum(nil))

	return err
}
