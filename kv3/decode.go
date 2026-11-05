package kv3

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

const (
	// Version 0, "VKV\x03" (see legacy.go).
	magicLegacy = 0x03564B56
	// "KV3" and the version in the low byte.
	magicPrefix = 0x4B563300

	// Ends the buffers and the binary blobs.
	trailer = 0xFFEEDD00

	// Far more than any file a mod has; it keeps a damaged size from
	// allocating gigabytes.
	maxSize  = 256 << 20
	maxDepth = 512
)

var (
	ErrNotKV3  = errors.New("kv3: not binary KV3")
	ErrVersion = errors.New("kv3: unsupported binary KV3: version")
	ErrCorrupt = errors.New("kv3: the data is damaged")
)

const (
	compressNone = 0
	compressLZ4  = 1
	compressZstd = 2
)

// The binary format's node types.
const (
	nodeNull           = 1
	nodeBool           = 2
	nodeInt64          = 3
	nodeUint64         = 4
	nodeDouble         = 5
	nodeString         = 6
	nodeBlob           = 7
	nodeArray          = 8
	nodeObject         = 9
	nodeArrayTyped     = 10
	nodeInt32          = 11
	nodeUint32         = 12
	nodeTrue           = 13
	nodeFalse          = 14
	nodeInt64Zero      = 15
	nodeInt64One       = 16
	nodeDoubleZero     = 17
	nodeDoubleOne      = 18
	nodeFloat          = 19
	nodeInt16          = 20
	nodeUint16         = 21
	nodeInt8           = 22
	nodeUint8          = 23
	nodeArrayByteLen   = 24 // a typed array whose length is one byte
	nodeArrayAuxiliary = 25 // the same, with its elements in the auxiliary buffer
	nodeUint32Zero     = 26
	nodeUint32One      = 27
)

var zstdDecoder = sync.OnceValue(func() *zstd.Decoder {
	d, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(maxSize))

	return d
})

type header struct {
	version     int
	format      GUID
	compression uint32
	frameSize   int

	countBytes1, countBytes2, countBytes4, countBytes8 int
	// Before version 5, the size of the strings and the types together.
	countTypes int

	sizeUncompressedTotal, sizeCompressedTotal int
	countBlobs, sizeBlobs                      int

	sizeUncompressed1, sizeCompressed1 int
	sizeUncompressed2, sizeCompressed2 int

	// Version 5 keeps the values in a second buffer.
	countBytes1Buffer2, countBytes2Buffer2, countBytes4Buffer2, countBytes8Buffer2 int
	countObjects                                                                   int
}

// corrupt is panicked with inside Decode and recovered there, so the many
// bounds checks don't each need an error return.
type corrupt string

func fail(format string, args ...any) {
	panic(corrupt(fmt.Sprintf(format, args...)))
}

// Decode decodes binary KV3, e.g. a compiled file's DATA block.
func Decode(data []byte) (doc *Document, err error) {
	if len(data) < 4 {
		return nil, ErrNotKV3
	}

	magic := binary.LittleEndian.Uint32(data)

	switch {
	case magic == magicLegacy:
		if len(data) < 20 {
			return nil, fmt.Errorf("%w: cut off", ErrCorrupt)
		}

		if !legacySupported(GUID(data[4:20])) {
			return nil, fmt.Errorf("%w: 0 with an encoding other than binary or LZ4", ErrVersion)
		}
	case magic&0xFFFFFF00 != magicPrefix:
		return nil, ErrNotKV3
	case magic&0xFF < 1 || magic&0xFF > 5:
		return nil, fmt.Errorf("%w: %d", ErrVersion, magic&0xFF)
	}

	defer func() {
		if r := recover(); r != nil {
			msg, ok := r.(corrupt)
			if !ok {
				panic(r)
			}

			doc, err = nil, fmt.Errorf("%w: %s", ErrCorrupt, msg)
		}
	}()

	in := cursor{b: data, pos: 4}

	if magic == magicLegacy {
		return decodeLegacy(&in), nil
	}

	h := readHeader(int(magic&0xFF), &in)

	return decodeBody(h, &in), nil
}

func readHeader(version int, in *cursor) header {
	h := header{version: version}
	h.format = GUID(in.take(16))
	h.compression = in.u32()

	if h.compression > compressZstd {
		fail("unknown compression %d", h.compression)
	}

	if version == 1 {
		h.countBytes1 = in.count()
		h.countBytes4 = in.count()
		h.countBytes8 = in.count()
		h.sizeUncompressedTotal = in.count()
		h.sizeCompressedTotal = len(in.b) - in.pos
	} else {
		if dictionary := in.u16(); dictionary != 0 {
			fail("compression dictionary %d", dictionary)
		}

		h.frameSize = int(in.u16())
		h.countBytes1 = in.count()
		h.countBytes4 = in.count()
		h.countBytes8 = in.count()
		h.countTypes = in.count()
		in.take(4) // how many objects and arrays: only hints
		h.sizeUncompressedTotal = in.count()
		h.sizeCompressedTotal = in.count()
		h.countBlobs = in.count()
		h.sizeBlobs = in.count()
	}

	if version >= 4 {
		h.countBytes2 = in.count()
		in.count() // the size of the blob frames' sizes, which the blob lengths imply
	}

	if version >= 5 {
		h.sizeUncompressed1 = in.count()
		h.sizeCompressed1 = in.count()
		h.sizeUncompressed2 = in.count()
		h.sizeCompressed2 = in.count()
		h.countBytes1Buffer2 = in.count()
		h.countBytes2Buffer2 = in.count()
		h.countBytes4Buffer2 = in.count()
		h.countBytes8Buffer2 = in.count()
		in.count() // nodes: a hint
		h.countObjects = in.count()
		in.take(8) // arrays and their elements: hints
	} else {
		h.sizeUncompressed1 = h.sizeUncompressedTotal
		h.sizeCompressed1 = h.sizeCompressedTotal
	}

	return h
}

// lanes are a buffer's values, kept apart by size.
type lanes struct {
	bytes1, bytes2, bytes4, bytes8 cursor
}

func (l *lanes) size() int {
	return len(l.bytes1.b) + len(l.bytes2.b) + len(l.bytes4.b) + len(l.bytes8.b)
}

func decodeBody(h header, in *cursor) *Document {
	d := &decoder{version: h.version}

	var blobSizes cursor

	// Buffer 1. Before version 5 it's the only one; in version 5 it holds
	// the strings and the auxiliary values.
	size1 := h.sizeUncompressed1

	// Before version 5, Zstandard compresses the buffer and the blobs as two
	// frames, and the compressed size covers both.
	if h.version < 5 && h.compression == compressZstd {
		size1 += h.sizeBlobs
	}

	raw1 := readBuffer(in, h.compression, h.sizeCompressed1, size1)
	buf1 := cursor{b: raw1[:h.sizeUncompressed1]}
	aux := readLanes(&buf1, h.countBytes1, h.countBytes2, h.countBytes4, h.countBytes8)

	if h.countBytes8 == 0 && h.version < 5 {
		// Before version 5 the lanes are aligned even when they're empty.
		buf1.align(8)
	}

	countStrings := aux.bytes4.count()

	if h.version >= 5 {
		d.strings = readStrings(&aux.bytes1, countStrings)
	} else {
		start := buf1.pos
		d.strings = readStrings(&buf1, countStrings)

		typesSize := h.countTypes - (buf1.pos - start)

		if h.version == 1 {
			typesSize = h.sizeUncompressedTotal - buf1.pos - 4
		}

		d.types = cursor{b: buf1.take(typesSize)}
		blobSizes = endOfBuffer(&buf1, h.countBlobs)
	}

	// Before version 5 there's only the one buffer.
	d.aux = &aux
	d.main = &aux

	// Buffer 2, in version 5: the lengths of objects, the values, and the
	// types.
	if h.version >= 5 {
		buf2 := cursor{b: readBuffer(in, h.compression, h.sizeCompressed2, h.sizeUncompressed2)}
		d.objectLengths = cursor{b: buf2.take(h.countObjects * 4)}

		main := readLanes(&buf2, h.countBytes1Buffer2, h.countBytes2Buffer2, h.countBytes4Buffer2, h.countBytes8Buffer2)
		d.main = &main
		d.types = cursor{b: buf2.take(h.countTypes)}
		blobSizes = endOfBuffer(&buf2, h.countBlobs)
	}

	if h.countBlobs > 0 {
		if h.version < 2 {
			fail("binary blobs in version 1")
		}

		d.blobs = readBlobs(h, in, &blobSizes, raw1[h.sizeUncompressed1:])
	}

	// Every value takes at least a byte, of its type or its data, except
	// the elements of a typed array of constants, e.g. nulls. This keeps a
	// damaged length from making millions of those.
	d.budget = len(d.types.b) + d.main.size() + d.aux.size() + len(d.blobs) + 1<<16

	t, flag := d.readType()

	return &Document{Format: h.format, Root: d.value(t, flag, d.main)}
}

func readBuffer(in *cursor, compression uint32, compressedSize, size int) []byte {
	if size > maxSize {
		fail("a buffer of %d bytes", size)
	}

	switch compression {
	case compressNone:
		return in.take(size)
	case compressLZ4:
		src := in.take(compressedSize)

		// LZ4 can't make anything more than 255 times smaller; a size past
		// that is damaged, and isn't worth allocating.
		if size > 255*len(src)+16 {
			fail("%d bytes of LZ4 data can't hold %d", len(src), size)
		}

		out := make([]byte, size)

		n, err := lz4.UncompressBlock(src, out)
		if err != nil || n != size {
			fail("LZ4 data doesn't decompress to %d bytes", size)
		}

		return out
	default:
		// Zstandard can shrink data any amount, so a damaged size is only
		// found out by decompressing; until then, don't trust it with more
		// than a megabyte.
		out, err := zstdDecoder().DecodeAll(in.take(compressedSize), make([]byte, 0, min(size, 1<<20)))
		if err != nil || len(out) != size {
			fail("Zstandard data doesn't decompress to %d bytes", size)
		}

		return out
	}
}

func readLanes(buf *cursor, count1, count2, count4, count8 int) lanes {
	var l lanes

	l.bytes1 = cursor{b: buf.take(count1)}

	if count2 > 0 {
		buf.align(2)
		l.bytes2 = cursor{b: buf.take(count2 * 2)}
	}

	if count4 > 0 {
		buf.align(4)
		l.bytes4 = cursor{b: buf.take(count4 * 4)}
	}

	if count8 > 0 {
		buf.align(8)
		l.bytes8 = cursor{b: buf.take(count8 * 8)}
	}

	return l
}

// endOfBuffer checks the end of the buffer that holds the types: the
// trailer, or with blobs, their lengths and what follows them, which it
// returns.
func endOfBuffer(buf *cursor, countBlobs int) cursor {
	if countBlobs == 0 {
		if buf.u32() != trailer {
			fail("no trailer after the types")
		}

		return cursor{}
	}

	return cursor{b: buf.b[buf.pos:]}
}

func readStrings(c *cursor, n int) []string {
	// Each takes at least its terminating zero.
	if n > len(c.b)-c.pos {
		fail("%d strings in %d bytes", n, len(c.b)-c.pos)
	}

	strings := make([]string, n)

	for i := range strings {
		end := c.pos

		for end < len(c.b) && c.b[end] != 0 {
			end++
		}

		if end == len(c.b) {
			fail("a string has no end")
		}

		strings[i] = string(c.b[c.pos:end])
		c.pos = end + 1
	}

	return strings
}

// readBlobs reads the binary blobs, which follow the buffers. sizes is
// what follows the types: the blobs' lengths, a trailer, and with LZ4 the
// compressed size of each frame. packed is what Zstandard decompressed
// after buffer 1 before version 5.
func readBlobs(h header, in *cursor, sizes *cursor, packed []byte) [][]byte {
	lengths := make([]int, h.countBlobs)
	total := 0

	for i := range lengths {
		lengths[i] = sizes.count()
		total += lengths[i]
	}

	if total != h.sizeBlobs || total > maxSize {
		fail("blob lengths add up to %d, not %d", total, h.sizeBlobs)
	}

	if sizes.u32() != trailer {
		fail("no trailer after the blob lengths")
	}

	var all []byte

	switch {
	case h.compression == compressNone:
		all = in.take(total)
	case h.compression == compressZstd && h.version < 5:
		all = packed
	case h.compression == compressZstd:
		all = readBuffer(in, compressZstd, h.sizeCompressedTotal-h.sizeCompressed1-h.sizeCompressed2, total)
	default:
		all = readLZ4Frames(h, in, sizes, lengths, total)
	}

	if len(all) != total {
		fail("blobs are %d bytes, not %d", len(all), total)
	}

	if in.u32() != trailer {
		fail("no trailer after the blobs")
	}

	blobs := make([][]byte, len(lengths))
	at := 0

	for i, n := range lengths {
		blobs[i] = all[at : at+n : at+n]
		at += n
	}

	return blobs
}

// readLZ4Frames decompresses blobs that LZ4 compressed in frames of up to
// frameSize bytes. A frame never spans two blobs, and frames can refer back
// to what came before them, even in earlier blobs, up to LZ4's 64 KB.
func readLZ4Frames(h header, in *cursor, sizes *cursor, lengths []int, total int) []byte {
	if h.frameSize == 0 {
		fail("LZ4 blobs without a frame size")
	}

	all := make([]byte, total)
	at := 0

	for _, n := range lengths {
		end := at + n

		for at < end {
			src := in.take(int(sizes.u16()))
			dst := all[at:min(at+h.frameSize, end)]
			dict := all[max(0, at-64<<10):at]

			got, err := lz4.UncompressBlockWithDict(src, dst, dict)
			if err != nil || got < 1 {
				fail("an LZ4 blob frame doesn't decompress")
			}

			at += got
		}
	}

	return all
}

type decoder struct {
	version int
	strings []string
	types   cursor
	// The values, and the auxiliary values, which only the elements of
	// typed arrays of nodeArrayAuxiliary use. Before version 5 they're the
	// same buffer.
	main, aux *lanes
	// In version 5 the lengths of objects have a buffer of their own.
	objectLengths cursor
	blobs         [][]byte
	nextBlob      int
	depth         int
	budget        int
}

func (d *decoder) readType() (byte, Flag) {
	t := d.types.u8()
	flag := NoFlag

	if d.version >= 3 {
		if t&0x80 != 0 {
			flag = Flag(d.types.u8())

			if flag == NoFlag || flag > FlagLocalize {
				fail("unknown flag %d", flag)
			}
		}

		// No known writer sets this; readers skip the byte that follows.
		if t&0x40 != 0 {
			d.types.u8()
		}

		return t & 0x3F, flag
	}

	if t&0x80 != 0 {
		flag = legacyFlag(d.types.u8())
	}

	return t & 0x7F, flag
}

// legacyFlag converts the bit field flags were before version 3. Only one
// is kept, the lowest; 16 marked multi-line strings and is dropped.
func legacyFlag(bits byte) Flag {
	switch {
	case bits&0x80 != 0:
		fail("unknown flags %#x", bits)
	case bits&1 != 0:
		return FlagResource
	case bits&2 != 0:
		return FlagResourceName
	case bits&8 != 0:
		return FlagPanorama
	case bits&16 != 0:
		return FlagSoundEvent
	case bits&32 != 0:
		return FlagSubclass
	}

	return NoFlag
}

func (d *decoder) string(index int32) string {
	// -1 is an empty string; so is any other index out of range.
	if index < 0 || int(index) >= len(d.strings) {
		return ""
	}

	return d.strings[index]
}

// value reads a value of node type t. Numbers come from l, which is the
// auxiliary lanes for the elements of a nodeArrayAuxiliary array; strings,
// blobs and lengths always come from the main lanes.
func (d *decoder) value(t byte, flag Flag, l *lanes) Value {
	d.budget--
	if d.budget < 0 {
		fail("more values than the data could hold")
	}

	v := Value{Flag: flag}
	m := d.main

	switch t {
	case nodeNull:
		v.Type = TypeNull
	case nodeTrue, nodeFalse:
		v.Type, v.Bool = TypeBool, t == nodeTrue
	case nodeInt64Zero, nodeInt64One:
		v.Type, v.Int = TypeInt64, int64(t-nodeInt64Zero)
	case nodeDoubleZero, nodeDoubleOne:
		v.Type, v.Float = TypeFloat64, float64(t-nodeDoubleZero)
	case nodeUint32Zero, nodeUint32One:
		v.Type, v.Uint = TypeUint32, uint64(t-nodeUint32Zero)
	case nodeBool:
		v.Type, v.Bool = TypeBool, l.bytes1.u8() != 0
	case nodeInt8:
		v.Type, v.Int = TypeInt8, int64(int8(l.bytes1.u8()))
	case nodeUint8:
		v.Type, v.Uint = TypeUint8, uint64(l.bytes1.u8())
	case nodeInt16:
		v.Type, v.Int = TypeInt16, int64(int16(l.bytes2.u16()))
	case nodeUint16:
		v.Type, v.Uint = TypeUint16, uint64(l.bytes2.u16())
	case nodeInt32:
		v.Type, v.Int = TypeInt32, int64(int32(l.bytes4.u32()))
	case nodeUint32:
		v.Type, v.Uint = TypeUint32, uint64(l.bytes4.u32())
	case nodeFloat:
		v.Type, v.Float = TypeFloat32, float64(math.Float32frombits(l.bytes4.u32()))
	case nodeInt64:
		v.Type, v.Int = TypeInt64, int64(l.bytes8.u64())
	case nodeUint64:
		v.Type, v.Uint = TypeUint64, l.bytes8.u64()
	case nodeDouble:
		v.Type, v.Float = TypeFloat64, math.Float64frombits(l.bytes8.u64())
	case nodeString:
		v.Type, v.String = TypeString, d.string(int32(m.bytes4.u32()))
	case nodeBlob:
		v.Type, v.Blob = TypeBlob, d.blob()
	case nodeArray:
		v.Type, v.Array = TypeArray, d.array()
	case nodeArrayTyped, nodeArrayByteLen, nodeArrayAuxiliary:
		v.Type, v.Array = TypeArray, d.typedArray(t)
	case nodeObject:
		v.Type, v.Object = TypeObject, d.object()
	default:
		fail("unknown node type %d", t)
	}

	return v
}

func (d *decoder) blob() []byte {
	if d.version < 2 {
		n := d.main.bytes4.count()

		return d.main.bytes1.take(n)
	}

	if d.nextBlob >= len(d.blobs) {
		fail("more blobs than the header says")
	}

	d.nextBlob++

	return d.blobs[d.nextBlob-1]
}

func (d *decoder) enter() {
	d.depth++
	if d.depth > maxDepth {
		fail("nested more than %d deep", maxDepth)
	}
}

func (d *decoder) array() []Value {
	d.enter()
	defer func() { d.depth-- }()

	n := d.main.bytes4.count()
	elems := make([]Value, 0, min(n, 1024))

	for range n {
		t, flag := d.readType()
		elems = append(elems, d.value(t, flag, d.main))
	}

	return elems
}

func (d *decoder) typedArray(t byte) []Value {
	d.enter()
	defer func() { d.depth-- }()

	var n int

	if t == nodeArrayTyped {
		n = d.main.bytes4.count()
	} else {
		n = int(d.main.bytes1.u8())
	}

	if n == 0 && d.version >= 2 {
		fail("an empty typed array")
	}

	elemType, flag := d.readType()

	l := d.main

	if t == nodeArrayAuxiliary {
		l = d.aux
	}

	elems := make([]Value, 0, min(n, 1024))

	for range n {
		elems = append(elems, d.value(elemType, flag, l))
	}

	return elems
}

func (d *decoder) object() Object {
	d.enter()
	defer func() { d.depth-- }()

	var n int

	if d.version >= 5 {
		n = d.objectLengths.count()
	} else {
		n = d.main.bytes4.count()
	}

	obj := make(Object, 0, min(n, 1024))

	// Only large objects get an index; for small ones a search is quicker.
	var index map[string]int

	if n > 16 {
		index = make(map[string]int, min(n, 1024))
	}

	for range n {
		t, flag := d.readType()
		key := d.string(int32(d.main.bytes4.u32()))
		v := d.value(t, flag, d.main)

		// A key that's there already replaces the earlier value, as
		// Source2Viewer has it.
		if i := find(obj, index, key); i >= 0 {
			obj[i].Value = v

			continue
		}

		if index != nil {
			index[key] = len(obj)
		}

		obj = append(obj, Member{Key: key, Value: v})
	}

	return obj
}

// find returns where key is in obj, looking it up in index if there is
// one, or -1.
func find(obj Object, index map[string]int, key string) int {
	if index == nil {
		return slices.IndexFunc(obj, func(m Member) bool { return m.Key == key })
	}

	if i, ok := index[key]; ok {
		return i
	}

	return -1
}

// A cursor reads little-endian values from b, failing past its end.
type cursor struct {
	b   []byte
	pos int
}

func (c *cursor) take(n int) []byte {
	if n < 0 || n > len(c.b)-c.pos {
		fail("%d bytes wanted at %d of %d", n, c.pos, len(c.b))
	}

	c.pos += n

	return c.b[c.pos-n : c.pos : c.pos]
}

func (c *cursor) align(n int) {
	pad := (n - c.pos%n) % n
	c.take(pad)
}

func (c *cursor) u8() byte    { return c.take(1)[0] }
func (c *cursor) u16() uint16 { return binary.LittleEndian.Uint16(c.take(2)) }
func (c *cursor) u32() uint32 { return binary.LittleEndian.Uint32(c.take(4)) }
func (c *cursor) u64() uint64 { return binary.LittleEndian.Uint64(c.take(8)) }

// count reads a size or count, which can't be negative and is checked
// against maxSize so arithmetic on it can't overflow.
func (c *cursor) count() int {
	n := int32(c.u32())
	if n < 0 || n > maxSize {
		fail("a count of %d", n)
	}

	return int(n)
}
