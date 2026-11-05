package kv3

import (
	"math"

	"github.com/pierrec/lz4/v4"
)

// Version 0 ("VKV\x03") names its encoding with a GUID. Some mods' particles
// and animation clips still have it, from older tools.
var (
	encodingBinary     = GUID{0x00, 0x05, 0x86, 0x1B, 0xD8, 0xF7, 0xC1, 0x40, 0xAD, 0x82, 0x75, 0xA4, 0x82, 0x67, 0xE7, 0x14}
	encodingBinaryLZ4  = GUID{0x8A, 0x34, 0x47, 0x68, 0xA1, 0x63, 0x5C, 0x4F, 0xA1, 0x97, 0x53, 0x80, 0x6F, 0xD9, 0xB1, 0x19}
	encodingBinaryZstd = GUID{0x00, 0x0A, 0x62, 0x6F, 0xF0, 0xFE, 0x05, 0x43, 0xA3, 0x5F, 0x04, 0x23, 0x46, 0xB1, 0xDB, 0x29}
	encodingBinaryAuto = GUID{0xE6, 0x09, 0xB1, 0x6E, 0x85, 0x6B, 0x83, 0x45, 0xA3, 0x12, 0x70, 0x3A, 0x6E, 0x04, 0x06, 0x8C}
)

// legacySupported says whether an encoding of version 0 can be decoded.
// Valve's own block compression, which no Deadlock file has been seen to
// use, can't.
func legacySupported(encoding GUID) bool {
	switch encoding {
	case encodingBinary, encodingBinaryLZ4, encodingBinaryZstd, encodingBinaryAuto:
		return true
	}

	return false
}

// Ends the data in version 0.
const legacyTrailer = 0xFFFFFFFF

// decodeLegacy decodes version 0, which keeps everything in one stream in
// the order it's read, rather than in lanes.
func decodeLegacy(in *cursor) *Document {
	encoding := GUID(in.take(16))
	format := GUID(in.take(16))

	var data []byte

	switch encoding {
	case encodingBinaryLZ4:
		size := in.count()
		src := in.take(len(in.b) - in.pos)

		if size > 255*len(src)+16 {
			fail("%d bytes of LZ4 data can't hold %d", len(src), size)
		}

		data = make([]byte, size)

		n, err := lz4.UncompressBlock(src, data)
		if err != nil || n != size {
			fail("LZ4 data doesn't decompress to %d bytes", size)
		}
	// Only block compression and LZ4 were ever written for version 0; the
	// other encodings it names are stored as they are.
	case encodingBinary, encodingBinaryZstd, encodingBinaryAuto:
		data = in.take(len(in.b) - in.pos)
	default:
		fail("unsupported encoding %v", encoding)
	}

	d := &legacyDecoder{in: cursor{b: data}}
	d.budget = len(data) + 1<<16

	n := d.in.count()
	d.strings = readStrings(&d.in, n)

	t, flag := d.readType()
	root := d.value(t, flag)

	if d.in.u32() != legacyTrailer {
		fail("no trailer after the data")
	}

	return &Document{Format: format, Root: root}
}

type legacyDecoder struct {
	in      cursor
	strings []string
	depth   int
	budget  int
}

func (d *legacyDecoder) readType() (byte, Flag) {
	t := d.in.u8()
	flag := NoFlag

	if t&0x80 != 0 {
		flag = legacyFlag(d.in.u8())
	}

	return t & 0x7F, flag
}

func (d *legacyDecoder) string() string {
	i := int32(d.in.u32())
	if i < 0 || int(i) >= len(d.strings) {
		return ""
	}

	return d.strings[i]
}

func (d *legacyDecoder) value(t byte, flag Flag) Value {
	d.budget--
	if d.budget < 0 {
		fail("more values than the data could hold")
	}

	v := Value{Flag: flag}

	switch t {
	case nodeNull:
		v.Type = TypeNull
	case nodeTrue, nodeFalse:
		v.Type, v.Bool = TypeBool, t == nodeTrue
	case nodeInt64Zero, nodeInt64One:
		v.Type, v.Int = TypeInt64, int64(t-nodeInt64Zero)
	case nodeDoubleZero, nodeDoubleOne:
		v.Type, v.Float = TypeFloat64, float64(t-nodeDoubleZero)
	case nodeBool:
		v.Type, v.Bool = TypeBool, d.in.u8() != 0
	case nodeInt32:
		v.Type, v.Int = TypeInt32, int64(int32(d.in.u32()))
	case nodeUint32:
		v.Type, v.Uint = TypeUint32, uint64(d.in.u32())
	case nodeInt64:
		v.Type, v.Int = TypeInt64, int64(d.in.u64())
	case nodeUint64:
		v.Type, v.Uint = TypeUint64, d.in.u64()
	case nodeDouble:
		v.Type, v.Float = TypeFloat64, math.Float64frombits(d.in.u64())
	case nodeString:
		v.Type, v.String = TypeString, d.string()
	case nodeBlob:
		v.Type, v.Blob = TypeBlob, d.in.take(d.in.count())
	case nodeArray, nodeArrayTyped:
		v.Type, v.Array = TypeArray, d.array(t == nodeArrayTyped)
	case nodeObject:
		v.Type, v.Object = TypeObject, d.object()
	default:
		fail("unknown node type %d", t)
	}

	return v
}

// array reads an array's elements, which each have a type of their own
// unless the array is typed, when one type comes first for all of them.
func (d *legacyDecoder) array(typed bool) []Value {
	d.depth++
	if d.depth > maxDepth {
		fail("nested more than %d deep", maxDepth)
	}

	defer func() { d.depth-- }()

	n := d.in.count()

	var (
		elemType byte
		flag     Flag
	)

	if typed {
		elemType, flag = d.readType()
	}

	elems := make([]Value, 0, min(n, 1024))

	for range n {
		if !typed {
			elemType, flag = d.readType()
		}

		elems = append(elems, d.value(elemType, flag))
	}

	return elems
}

func (d *legacyDecoder) object() Object {
	d.depth++
	if d.depth > maxDepth {
		fail("nested more than %d deep", maxDepth)
	}

	defer func() { d.depth-- }()

	n := d.in.count()
	obj := make(Object, 0, min(n, 1024))

	var index map[string]int

	if n > 16 {
		index = make(map[string]int, min(n, 1024))
	}

	for range n {
		// Unlike later versions, the key comes before the type.
		key := d.string()
		t, flag := d.readType()
		v := d.value(t, flag)

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
