package kv3

import (
	"encoding/binary"
	"errors"
	"math"
)

// ErrBlob is returned by Encode for a document with a binary blob, which
// it can't write yet; no table a mod changes has one.
var ErrBlob = errors.New("kv3: can't write binary blobs")

// Encode writes doc as binary KV3 version 5, uncompressed: the way the
// game's own uncompressed files are written, which the game reads.
func Encode(doc *Document) ([]byte, error) {
	e := &encoder{index: map[string]int32{}}

	if err := e.value(doc.Root); err != nil {
		return nil, err
	}

	le := binary.LittleEndian

	// Buffer 1: the strings, then their count.
	buf1 := append([]byte(nil), e.strings...)
	buf1 = pad(buf1, 4)
	buf1 = le.AppendUint32(buf1, uint32(e.count))

	// Buffer 2: the objects' lengths, the values by size, the types.
	buf2 := append([]byte(nil), e.objectLengths...)
	buf2 = append(buf2, e.bytes1...)

	if len(e.bytes2) > 0 {
		buf2 = pad(buf2, 2)
		buf2 = append(buf2, e.bytes2...)
	}

	if len(e.bytes4) > 0 {
		buf2 = pad(buf2, 4)
		buf2 = append(buf2, e.bytes4...)
	}

	if len(e.bytes8) > 0 {
		buf2 = pad(buf2, 8)
		buf2 = append(buf2, e.bytes8...)
	}

	buf2 = append(buf2, e.types...)
	buf2 = le.AppendUint32(buf2, trailer)

	total := len(buf1) + len(buf2)

	out := le.AppendUint32(nil, magicPrefix|5)
	out = append(out, doc.Format[:]...)
	out = le.AppendUint32(out, compressNone)
	out = le.AppendUint16(out, 0) // compression dictionary
	out = le.AppendUint16(out, 0) // frame size: only for LZ4
	out = le.AppendUint32(out, uint32(len(e.strings)))
	out = le.AppendUint32(out, 1) // 4-byte values in buffer 1: the string count
	out = le.AppendUint32(out, 0)
	out = le.AppendUint32(out, uint32(len(e.types)))
	out = le.AppendUint16(out, uint16(min(e.objects, math.MaxUint16)))
	out = le.AppendUint16(out, uint16(min(e.arrays, math.MaxUint16)))
	out = le.AppendUint32(out, uint32(total))
	out = le.AppendUint32(out, uint32(total)) // "compressed", as the game's uncompressed files have it
	out = le.AppendUint32(out, 0)             // blobs
	out = le.AppendUint32(out, 0)             // their size
	out = le.AppendUint32(out, 0)             // 2-byte values in buffer 1
	out = le.AppendUint32(out, 0)             // the size of the blob frames' sizes
	out = le.AppendUint32(out, uint32(len(buf1)))
	out = le.AppendUint32(out, 0) // its compressed size: 0 when it isn't
	out = le.AppendUint32(out, uint32(len(buf2)))
	out = le.AppendUint32(out, 0)
	out = le.AppendUint32(out, uint32(len(e.bytes1)))
	out = le.AppendUint32(out, uint32(len(e.bytes2)/2))
	out = le.AppendUint32(out, uint32(len(e.bytes4)/4))
	out = le.AppendUint32(out, uint32(len(e.bytes8)/8))
	out = le.AppendUint32(out, uint32(e.nodes))
	out = le.AppendUint32(out, uint32(e.objects))
	out = le.AppendUint32(out, uint32(e.arrays))
	out = le.AppendUint32(out, uint32(e.elements))

	out = append(out, buf1...)
	out = append(out, buf2...)

	return out, nil
}

func pad(b []byte, n int) []byte {
	for len(b)%n != 0 {
		b = append(b, 0)
	}

	return b
}

type encoder struct {
	strings []byte
	count   int32
	index   map[string]int32

	types                          []byte
	objectLengths                  []byte
	bytes1, bytes2, bytes4, bytes8 []byte

	// Counts the header gives as hints.
	nodes, objects, arrays, elements int
}

func (e *encoder) str(s string) int32 {
	// The game writes an empty string as -1, not as a string.
	if s == "" {
		return -1
	}

	i, ok := e.index[s]
	if !ok {
		i = e.count
		e.index[s] = i
		e.count++
		e.strings = append(e.strings, s...)
		e.strings = append(e.strings, 0)
	}

	return i
}

func (e *encoder) typ(t byte, flag Flag) {
	if flag != NoFlag {
		e.types = append(e.types, t|0x80, byte(flag))
	} else {
		e.types = append(e.types, t)
	}
}

func (e *encoder) put4(v uint32) { e.bytes4 = binary.LittleEndian.AppendUint32(e.bytes4, v) }
func (e *encoder) put8(v uint64) { e.bytes8 = binary.LittleEndian.AppendUint64(e.bytes8, v) }

func (e *encoder) value(v Value) error {
	e.nodes++

	switch v.Type {
	case TypeNull:
		e.typ(nodeNull, v.Flag)
	case TypeBool:
		if v.Bool {
			e.typ(nodeTrue, v.Flag)
		} else {
			e.typ(nodeFalse, v.Flag)
		}
	case TypeInt64:
		switch v.Int {
		case 0:
			e.typ(nodeInt64Zero, v.Flag)
		case 1:
			e.typ(nodeInt64One, v.Flag)
		default:
			e.typ(nodeInt64, v.Flag)
			e.put8(uint64(v.Int))
		}
	case TypeUint64:
		e.typ(nodeUint64, v.Flag)
		e.put8(v.Uint)
	case TypeFloat64:
		switch v.Float {
		case 0:
			e.typ(nodeDoubleZero, v.Flag)
		case 1:
			e.typ(nodeDoubleOne, v.Flag)
		default:
			e.typ(nodeDouble, v.Flag)
			e.put8(math.Float64bits(v.Float))
		}
	case TypeInt32:
		e.typ(nodeInt32, v.Flag)
		e.put4(uint32(int32(v.Int)))
	case TypeUint32:
		switch v.Uint {
		case 0:
			e.typ(nodeUint32Zero, v.Flag)
		case 1:
			e.typ(nodeUint32One, v.Flag)
		default:
			e.typ(nodeUint32, v.Flag)
			e.put4(uint32(v.Uint))
		}
	case TypeFloat32:
		e.typ(nodeFloat, v.Flag)
		e.put4(math.Float32bits(float32(v.Float)))
	case TypeInt16:
		e.typ(nodeInt16, v.Flag)
		e.bytes2 = binary.LittleEndian.AppendUint16(e.bytes2, uint16(int16(v.Int)))
	case TypeUint16:
		e.typ(nodeUint16, v.Flag)
		e.bytes2 = binary.LittleEndian.AppendUint16(e.bytes2, uint16(v.Uint))
	case TypeInt8:
		e.typ(nodeInt8, v.Flag)
		e.bytes1 = append(e.bytes1, byte(int8(v.Int)))
	case TypeUint8:
		e.typ(nodeUint8, v.Flag)
		e.bytes1 = append(e.bytes1, byte(v.Uint))
	case TypeString:
		e.typ(nodeString, v.Flag)
		e.put4(uint32(e.str(v.String)))
	case TypeBlob:
		return ErrBlob
	case TypeArray:
		e.typ(nodeArray, v.Flag)
		e.put4(uint32(len(v.Array)))
		e.arrays++
		e.elements += len(v.Array)

		for _, elem := range v.Array {
			if err := e.value(elem); err != nil {
				return err
			}
		}
	case TypeObject:
		e.typ(nodeObject, v.Flag)
		e.objectLengths = binary.LittleEndian.AppendUint32(e.objectLengths, uint32(len(v.Object)))
		e.objects++

		for _, m := range v.Object {
			// The key goes in the same lane as the value's 4-byte data, so
			// it has to come first.
			e.put4(uint32(e.str(m.Key)))

			if err := e.value(m.Value); err != nil {
				return err
			}
		}
	}

	return nil
}
