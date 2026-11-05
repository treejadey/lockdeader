package kv3

import (
	"encoding/binary"
	"math"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// A builder lays out binary KV3 the way Source 2's resource compiler does,
// one node at a time, so tests can make any version and any node type
// without game files.
type builder struct {
	version     int
	compression uint32
	// For LZ4 blobs: how big a frame is.
	frameSize int

	strings []string
	index   map[string]int32
	types   []byte
	flag    Flag

	main, aux    lanes
	objectLength []byte // version 5
	blobs        [][]byte
}

func newBuilder(version int, compression uint32) *builder {
	return &builder{version: version, compression: compression, frameSize: 16384, index: map[string]int32{}}
}

var le = binary.LittleEndian

func (l *lanes) put1(b byte)     { l.bytes1.b = append(l.bytes1.b, b) }
func (l *lanes) put2(v uint16)   { l.bytes2.b = le.AppendUint16(l.bytes2.b, v) }
func (l *lanes) put4(v uint32)   { l.bytes4.b = le.AppendUint32(l.bytes4.b, v) }
func (l *lanes) put8(v uint64)   { l.bytes8.b = le.AppendUint64(l.bytes8.b, v) }
func (l *lanes) putI4(v int32)   { l.put4(uint32(v)) }
func (l *lanes) putF4(v float32) { l.put4(math.Float32bits(v)) }

// auxLanes is where the elements of an auxiliary array go: a buffer of
// their own in version 5, the same as everything else before it.
func (b *builder) auxLanes() *lanes {
	if b.version >= 5 {
		return &b.aux
	}

	return &b.main
}

func (b *builder) str(s string) int32 {
	i, ok := b.index[s]
	if !ok {
		i = int32(len(b.strings))
		b.index[s] = i
		b.strings = append(b.strings, s)
	}

	return i
}

// flagged sets the flag of the next node.
func (b *builder) flagged(f Flag) *builder {
	b.flag = f

	return b
}

func (b *builder) typ(t byte) {
	f := b.flag
	b.flag = NoFlag

	switch {
	case f == NoFlag:
		b.types = append(b.types, t)
	case b.version >= 3:
		b.types = append(b.types, t|0x80, byte(f))
	default:
		bits := map[Flag]byte{FlagResource: 1, FlagResourceName: 2, FlagPanorama: 8, FlagSoundEvent: 16, FlagSubclass: 32}[f]
		b.types = append(b.types, t|0x80, bits)
	}
}

// key starts an object's member: its type comes next.
func (b *builder) key(k string) *builder {
	b.main.putI4(b.str(k))

	return b
}

func (b *builder) node(t byte) { b.typ(t) }

// Each writes a node's type, then its value to the lane of its size.
func (b *builder) int32(v int32)    { b.typ(nodeInt32); b.main.putI4(v) }
func (b *builder) uint32(v uint32)  { b.typ(nodeUint32); b.main.put4(v) }
func (b *builder) int64(v int64)    { b.typ(nodeInt64); b.main.put8(uint64(v)) }
func (b *builder) uint64(v uint64)  { b.typ(nodeUint64); b.main.put8(v) }
func (b *builder) int8(v int8)      { b.typ(nodeInt8); b.main.put1(byte(v)) }
func (b *builder) uint8(v uint8)    { b.typ(nodeUint8); b.main.put1(v) }
func (b *builder) int16(v int16)    { b.typ(nodeInt16); b.main.put2(uint16(v)) }
func (b *builder) uint16(v uint16)  { b.typ(nodeUint16); b.main.put2(v) }
func (b *builder) float(v float32)  { b.typ(nodeFloat); b.main.putF4(v) }
func (b *builder) double(v float64) { b.typ(nodeDouble); b.main.put8(math.Float64bits(v)) }

func (b *builder) boolean(v bool) {
	b.typ(nodeBool)

	if v {
		b.main.put1(1)
	} else {
		b.main.put1(0)
	}
}

func (b *builder) string(s string)     { b.typ(nodeString); b.main.putI4(b.str(s)) }
func (b *builder) stringIndex(i int32) { b.typ(nodeString); b.main.putI4(i) }

func (b *builder) object(n int) {
	b.typ(nodeObject)

	if b.version >= 5 {
		b.objectLength = le.AppendUint32(b.objectLength, uint32(n))
	} else {
		b.main.putI4(int32(n))
	}
}

func (b *builder) array(n int) {
	b.typ(nodeArray)
	b.main.putI4(int32(n))
}

// typedArray starts an array whose elements are all of type elem; they
// follow without types of their own.
func (b *builder) typedArray(n int, elem byte) {
	b.typ(nodeArrayTyped)
	b.main.putI4(int32(n))
	b.typ(elem)
}

func (b *builder) byteLengthArray(n int, elem byte) {
	b.typ(nodeArrayByteLen)
	b.main.put1(byte(n))
	b.typ(elem)
}

// auxArray starts an array whose elements are in the auxiliary lanes
// (auxLanes).
func (b *builder) auxArray(n int, elem byte) {
	b.typ(nodeArrayAuxiliary)
	b.main.put1(byte(n))
	b.typ(elem)
}

func (b *builder) blob(data []byte) {
	b.typ(nodeBlob)

	if b.version < 2 {
		b.main.putI4(int32(len(data)))
		b.main.bytes1.b = append(b.main.bytes1.b, data...)

		return
	}

	b.blobs = append(b.blobs, data)
}

func align(buf []byte, n int) []byte {
	for len(buf)%n != 0 {
		buf = append(buf, 0)
	}

	return buf
}

func (b *builder) stringBytes() []byte {
	var out []byte

	for _, s := range b.strings {
		out = append(out, s...)
		out = append(out, 0)
	}

	return out
}

// layLanes lays out lanes as the format does: 1-byte values, then 2, 4
// and 8, each aligned to its size, leaving out empty ones.
func layLanes(buf []byte, l lanes, pre4 []byte, alignEmpty bool) []byte {
	buf = append(buf, l.bytes1.b...)

	if len(l.bytes2.b) > 0 {
		buf = align(buf, 2)
		buf = append(buf, l.bytes2.b...)
	}

	if bytes4 := append(pre4, l.bytes4.b...); len(bytes4) > 0 {
		buf = align(buf, 4)
		buf = append(buf, bytes4...)
	}

	if len(l.bytes8.b) > 0 || alignEmpty {
		buf = align(buf, 8)
		buf = append(buf, l.bytes8.b...)
	}

	return buf
}

func (b *builder) compress(data []byte) []byte {
	switch b.compression {
	case compressLZ4:
		return lz4Block(data)
	case compressZstd:
		w, _ := zstd.NewWriter(nil)
		defer w.Close()

		return w.EncodeAll(data, nil)
	}

	return data
}

// lz4Block compresses data as one LZ4 block, or if LZ4 can't make it
// smaller, stores it as one run of literals, which is still a valid block.
func lz4Block(data []byte) []byte {
	out := make([]byte, lz4.CompressBlockBound(len(data)))

	n, _ := lz4.CompressBlock(data, out, nil)
	if n > 0 {
		return out[:n]
	}

	return lz4Literals(data)
}

func lz4Literals(data []byte) []byte {
	n := len(data)

	if n < 15 {
		return append([]byte{byte(n << 4)}, data...)
	}

	out := []byte{0xF0}

	for rest := n - 15; ; rest -= 255 {
		if rest < 255 {
			out = append(out, byte(rest))

			break
		}

		out = append(out, 255)
	}

	return append(out, data...)
}

// blobSection is what follows the types when there are blobs: their
// lengths and a trailer, and for LZ4 the size of each compressed frame; and
// the blobs' data, which follows the buffers.
func (b *builder) blobSection() (sizes, data []byte) {
	if len(b.blobs) == 0 {
		return le.AppendUint32(nil, trailer), nil
	}

	for _, blob := range b.blobs {
		sizes = le.AppendUint32(sizes, uint32(len(blob)))
	}

	sizes = le.AppendUint32(sizes, trailer)

	var all []byte

	for _, blob := range b.blobs {
		all = append(all, blob...)
	}

	switch b.compression {
	case compressNone:
		data = all
	case compressLZ4:
		for _, blob := range b.blobs {
			for at := 0; at < len(blob); at += b.frameSize {
				frame := lz4Literals(blob[at:min(at+b.frameSize, len(blob))])
				sizes = le.AppendUint16(sizes, uint16(len(frame)))
				data = append(data, frame...)
			}
		}
	case compressZstd:
		data = b.compress(all)
	}

	return sizes, data
}

func (b *builder) totalBlobs() int {
	n := 0

	for _, blob := range b.blobs {
		n += len(blob)
	}

	return n
}

// encode lays out everything added so far.
func (b *builder) encode() []byte {
	if b.version >= 5 {
		return b.encode5()
	}

	strs := b.stringBytes()
	sizes, blobData := b.blobSection()

	buf := layLanes(nil, b.main, le.AppendUint32(nil, uint32(len(b.strings))), true)
	buf = append(buf, strs...)
	buf = append(buf, b.types...)

	var compressed []byte

	if b.compression == compressZstd {
		buf = append(buf, sizes...)
		// The buffer and the blobs are two frames, together.
		compressed = append(b.compress(buf), blobData...)
		blobData = nil
	} else {
		buf = append(buf, sizes...)
		compressed = b.compress(buf)
	}

	out := le.AppendUint32(nil, magicPrefix|uint32(b.version))
	out = append(out, FormatGeneric[:]...)
	out = le.AppendUint32(out, b.compression)

	count4 := 1 + len(b.main.bytes4.b)/4

	if b.version == 1 {
		out = le.AppendUint32(out, uint32(len(b.main.bytes1.b)))
		out = le.AppendUint32(out, uint32(count4))
		out = le.AppendUint32(out, uint32(len(b.main.bytes8.b)/8))
		out = le.AppendUint32(out, uint32(len(buf)))
	} else {
		out = b.header2to4(out, len(buf), len(compressed), len(strs), count4)
	}

	out = append(out, compressed...)

	if len(b.blobs) > 0 {
		out = append(out, blobData...)
		out = le.AppendUint32(out, trailer)
	}

	return out
}

func (b *builder) frameSizeField() uint16 {
	if b.compression == compressLZ4 {
		return uint16(b.frameSize)
	}

	return 0
}

func (b *builder) header2to4(out []byte, size, compressed, strs, count4 int) []byte {
	out = le.AppendUint16(out, 0)
	out = le.AppendUint16(out, b.frameSizeField())
	out = le.AppendUint32(out, uint32(len(b.main.bytes1.b)))
	out = le.AppendUint32(out, uint32(count4))
	out = le.AppendUint32(out, uint32(len(b.main.bytes8.b)/8))
	out = le.AppendUint32(out, uint32(strs+len(b.types)))
	out = le.AppendUint32(out, 0) // objects and arrays: hints
	out = le.AppendUint32(out, uint32(size))
	out = le.AppendUint32(out, uint32(compressed))
	out = le.AppendUint32(out, uint32(len(b.blobs)))
	out = le.AppendUint32(out, uint32(b.totalBlobs()))

	if b.version >= 4 {
		out = le.AppendUint32(out, uint32(len(b.main.bytes2.b)/2))
		out = le.AppendUint32(out, 0)
	}

	return out
}

func (b *builder) encode5() []byte {
	strs := b.stringBytes()
	sizes, blobData := b.blobSection()

	// Buffer 1: the strings, then the auxiliary values.
	aux := b.aux
	aux.bytes1.b = append(strs, aux.bytes1.b...)
	buf1 := layLanes(nil, aux, le.AppendUint32(nil, uint32(len(b.strings))), false)

	// Buffer 2: the objects' lengths, the values, the types.
	buf2 := append([]byte(nil), b.objectLength...)
	buf2 = layLanes(buf2, b.main, nil, false)
	buf2 = append(buf2, b.types...)
	buf2 = append(buf2, sizes...)

	c1, c2 := b.compress(buf1), b.compress(buf2)

	out := le.AppendUint32(nil, magicPrefix|5)
	out = append(out, FormatGeneric[:]...)
	out = le.AppendUint32(out, b.compression)
	out = le.AppendUint16(out, 0)
	out = le.AppendUint16(out, b.frameSizeField())
	out = le.AppendUint32(out, uint32(len(aux.bytes1.b)))
	out = le.AppendUint32(out, uint32(1+len(aux.bytes4.b)/4))
	out = le.AppendUint32(out, uint32(len(aux.bytes8.b)/8))
	out = le.AppendUint32(out, uint32(len(b.types)))
	out = le.AppendUint32(out, 0) // objects and arrays: hints
	out = le.AppendUint32(out, uint32(len(buf1)+len(buf2)))
	out = le.AppendUint32(out, uint32(len(c1)+len(c2)+len(blobData)))
	out = le.AppendUint32(out, uint32(len(b.blobs)))
	out = le.AppendUint32(out, uint32(b.totalBlobs()))
	out = le.AppendUint32(out, uint32(len(aux.bytes2.b)/2))
	out = le.AppendUint32(out, 0) // the size of the frame sizes
	out = le.AppendUint32(out, uint32(len(buf1)))
	out = le.AppendUint32(out, uint32(len(c1)))
	out = le.AppendUint32(out, uint32(len(buf2)))
	out = le.AppendUint32(out, uint32(len(c2)))
	out = le.AppendUint32(out, uint32(len(b.main.bytes1.b)))
	out = le.AppendUint32(out, uint32(len(b.main.bytes2.b)/2))
	out = le.AppendUint32(out, uint32(len(b.main.bytes4.b)/4))
	out = le.AppendUint32(out, uint32(len(b.main.bytes8.b)/8))
	out = le.AppendUint32(out, 0) // nodes: a hint
	out = le.AppendUint32(out, uint32(len(b.objectLength)/4))
	out = le.AppendUint32(out, 0) // arrays: a hint
	out = le.AppendUint32(out, 0) // array elements: a hint

	out = append(out, c1...)
	out = append(out, c2...)

	if len(b.blobs) > 0 {
		out = append(out, blobData...)
		out = le.AppendUint32(out, trailer)
	}

	return out
}

// legacy lays out version 0, where each value follows its type in one
// stream, and an object's members put their key before their type.
type legacy struct {
	strings []string
	index   map[string]uint32
	body    []byte
}

func newLegacy() *legacy {
	return &legacy{index: map[string]uint32{}}
}

func (l *legacy) str(s string) uint32 {
	i, ok := l.index[s]
	if !ok {
		i = uint32(len(l.strings))
		l.index[s] = i
		l.strings = append(l.strings, s)
	}

	return i
}

func (l *legacy) key(k string) *legacy {
	l.body = le.AppendUint32(l.body, l.str(k))

	return l
}

// typ writes a type, with a flag as the bit field it was then.
func (l *legacy) typ(t byte, bits byte) {
	if bits != 0 {
		l.body = append(l.body, t|0x80, bits)
	} else {
		l.body = append(l.body, t)
	}
}

func (l *legacy) int32(v int32) {
	l.typ(nodeInt32, 0)
	l.body = le.AppendUint32(l.body, uint32(v))
}

func (l *legacy) double(v float64) {
	l.typ(nodeDouble, 0)
	l.body = le.AppendUint64(l.body, math.Float64bits(v))
}

func (l *legacy) string(s string, bits byte) {
	l.typ(nodeString, bits)
	l.body = le.AppendUint32(l.body, l.str(s))
}

func (l *legacy) count(t byte, n int) {
	l.typ(t, 0)
	l.body = le.AppendUint32(l.body, uint32(n))
}

func (l *legacy) encode(encoding GUID) []byte {
	data := le.AppendUint32(nil, uint32(len(l.strings)))

	for _, s := range l.strings {
		data = append(data, s...)
		data = append(data, 0)
	}

	data = append(data, l.body...)
	data = le.AppendUint32(data, legacyTrailer)

	out := le.AppendUint32(nil, magicLegacy)
	out = append(out, encoding[:]...)
	out = append(out, FormatGeneric[:]...)

	if encoding == encodingBinaryLZ4 {
		out = le.AppendUint32(out, uint32(len(data)))
		data = lz4Block(data)
	}

	return append(out, data...)
}
