package kv3

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type variant struct {
	version     int
	compression uint32
}

func (v variant) String() string {
	return fmt.Sprintf("v%d/%s", v.version, []string{"none", "lz4", "zstd"}[v.compression])
}

// variants is every version and compression the format has: version 1
// only knew LZ4.
func variants() []variant {
	var all []variant

	for version := 1; version <= 5; version++ {
		for c := uint32(compressNone); c <= compressZstd; c++ {
			if version == 1 && c == compressZstd {
				continue
			}

			all = append(all, variant{version, c})
		}
	}

	return all
}

func str(s string) Value              { return Value{Type: TypeString, String: s} }
func i32(n int64) Value               { return Value{Type: TypeInt32, Int: n} }
func obj(members ...Member) Value     { return Value{Type: TypeObject, Object: members} }
func arr(elems ...Value) Value        { return Value{Type: TypeArray, Array: elems} }
func member(k string, v Value) Member { return Member{Key: k, Value: v} }

// soundEvents builds a document like the sound event tables mods change,
// with every kind of node that all versions have.
func soundEvents(b *builder) Value {
	b.object(9)

	b.key("Generated.Fencer.Hero.Pick.VO").object(4)
	b.key("base").string("Base.HeroPick.VO.2d")
	b.key("vsnd_files").array(2)
	b.flagged(FlagResourceName).string("sounds/vo/fencer/fencer_select_01.vsnd")
	b.string("sounds/vo/fencer/fencer_select_02.vsnd")
	b.key("vsnd_duration").double(3.822687)
	b.key("context").flagged(FlagSoundEvent).string("hero_fencer")

	b.key("volume").uint32(4000000000)
	b.key("big").int64(-1 << 40)
	b.key("huge").uint64(1 << 63)
	b.key("on").boolean(true)
	b.key("nothing").node(nodeNull)
	b.key("constants").array(4)
	b.node(nodeInt64One)
	b.node(nodeDoubleZero)
	b.node(nodeFalse)
	b.node(nodeUint32One)
	b.key("ints").typedArray(3, nodeInt32)
	b.main.putI4(1)
	b.main.putI4(-2)
	b.main.putI4(3)
	b.key("missing").stringIndex(-1)

	return obj(
		member("Generated.Fencer.Hero.Pick.VO", obj(
			member("base", str("Base.HeroPick.VO.2d")),
			member("vsnd_files", arr(
				Value{Type: TypeString, Flag: FlagResourceName, String: "sounds/vo/fencer/fencer_select_01.vsnd"},
				str("sounds/vo/fencer/fencer_select_02.vsnd"),
			)),
			member("vsnd_duration", Value{Type: TypeFloat64, Float: 3.822687}),
			member("context", Value{Type: TypeString, Flag: FlagSoundEvent, String: "hero_fencer"}),
		)),
		member("volume", Value{Type: TypeUint32, Uint: 4000000000}),
		member("big", Value{Type: TypeInt64, Int: -1 << 40}),
		member("huge", Value{Type: TypeUint64, Uint: 1 << 63}),
		member("on", Value{Type: TypeBool, Bool: true}),
		member("nothing", Value{Type: TypeNull}),
		member("constants", arr(
			Value{Type: TypeInt64, Int: 1},
			Value{Type: TypeFloat64},
			Value{Type: TypeBool},
			Value{Type: TypeUint32, Uint: 1},
		)),
		member("ints", arr(i32(1), i32(-2), i32(3))),
		member("missing", str("")),
	)
}

func decode(t *testing.T, data []byte) *Document {
	t.Helper()

	doc, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	return doc
}

func TestDecodeEveryVersion(t *testing.T) {
	for _, v := range variants() {
		t.Run(v.String(), func(t *testing.T) {
			b := newBuilder(v.version, v.compression)
			want := soundEvents(b)

			doc := decode(t, b.encode())

			if doc.Format != FormatGeneric {
				t.Errorf("Format = %v, want generic", doc.Format)
			}

			if !reflect.DeepEqual(doc.Root, want) {
				t.Errorf("Root =\n%s\nwant\n%s", (&Document{Root: doc.Root}).Text(), (&Document{Root: want}).Text())
			}
		})
	}
}

// Version 4 added numbers smaller than 32 bits, and arrays with a byte for
// their length; version 5 put some arrays' elements in a buffer of their
// own.
func TestDecodeSmallNumbersAndAuxiliaryArrays(t *testing.T) {
	for _, v := range variants() {
		if v.version < 4 {
			continue
		}

		t.Run(v.String(), func(t *testing.T) {
			b := newBuilder(v.version, v.compression)
			b.object(7)
			b.key("i8").int8(-5)
			b.key("u8").uint8(250)
			b.key("i16").int16(-30000)
			b.key("u16").uint16(60000)
			b.key("f").float(0.25)
			b.key("bytes").byteLengthArray(2, nodeUint8)
			b.main.put1(7)
			b.main.put1(8)
			b.key("floats").auxArray(3, nodeFloat)

			aux := b.auxLanes()
			aux.putF4(1.5)
			aux.putF4(-2)
			aux.putF4(0)

			want := obj(
				member("i8", Value{Type: TypeInt8, Int: -5}),
				member("u8", Value{Type: TypeUint8, Uint: 250}),
				member("i16", Value{Type: TypeInt16, Int: -30000}),
				member("u16", Value{Type: TypeUint16, Uint: 60000}),
				member("f", Value{Type: TypeFloat32, Float: 0.25}),
				member("bytes", arr(Value{Type: TypeUint8, Uint: 7}, Value{Type: TypeUint8, Uint: 8})),
				member("floats", arr(
					Value{Type: TypeFloat32, Float: 1.5},
					Value{Type: TypeFloat32, Float: -2},
					Value{Type: TypeFloat32, Float: 0},
				)),
			)

			if got := decode(t, b.encode()).Root; !reflect.DeepEqual(got, want) {
				t.Errorf("Root =\n%s\nwant\n%s", (&Document{Root: got}).Text(), (&Document{Root: want}).Text())
			}
		})
	}
}

func TestDecodeBlobs(t *testing.T) {
	first := bytes.Repeat([]byte("0123456789"), 5)
	second := []byte("tiny")

	for _, v := range variants() {
		t.Run(v.String(), func(t *testing.T) {
			b := newBuilder(v.version, v.compression)
			// Several LZ4 frames to a blob.
			b.frameSize = 16
			b.array(3)
			b.blob(first)
			b.blob(second)
			b.blob(nil)

			got := decode(t, b.encode()).Root

			want := arr(Value{Type: TypeBlob, Blob: first}, Value{Type: TypeBlob, Blob: second}, Value{Type: TypeBlob, Blob: []byte{}})

			if len(got.Array) != 3 {
				t.Fatalf("Root = %+v, want 3 blobs", got)
			}

			for i := range want.Array {
				if g := got.Array[i]; g.Type != TypeBlob || !bytes.Equal(g.Blob, want.Array[i].Blob) {
					t.Errorf("blob %d = %v %q, want %q", i, g.Type, g.Blob, want.Array[i].Blob)
				}
			}
		})
	}
}

// LZ4 frames of blobs can copy from earlier frames, even of an earlier
// blob, as LZ4's chained blocks do.
func TestDecodeLZ4FramesReferBack(t *testing.T) {
	b := newBuilder(5, compressLZ4)
	b.array(2)
	b.blob([]byte("abcdefgh"))
	b.blob([]byte("cdefgh"))

	data := b.encode()

	// Replace the second blob's frame, stored as literals, with one that
	// copies its 6 bytes from 6 back, in the first blob.
	frame := lz4Literals([]byte("cdefgh"))
	ref := []byte{0x02, 6, 0} // no literals, a match of 4+2 bytes, offset 6

	at := bytes.LastIndex(data, frame)
	if at < 0 {
		t.Fatal("no literal frame to replace")
	}

	data = append(append(append([]byte(nil), data[:at]...), ref...), data[at+len(frame):]...)

	// The frame's size, in the list of sizes, changes too.
	sizeAt := bytes.LastIndex(data[:at], []byte{byte(len(frame)), 0})
	data[sizeAt] = byte(len(ref))

	// And so does the compressed total, in the header.
	total := le.Uint32(data[52:])
	le.PutUint32(data[52:], total-uint32(len(frame)-len(ref)))

	got := decode(t, data).Root

	if len(got.Array) != 2 || string(got.Array[1].Blob) != "cdefgh" {
		t.Errorf("Root = %+v, want the second blob copied from the first", got)
	}
}

// A key that's there already replaces its value, keeping its place.
func TestDecodeRepeatedKey(t *testing.T) {
	for _, n := range []int{3, 30} {
		b := newBuilder(5, compressNone)
		b.object(n)
		b.key("a").int32(1)
		b.key("b").int32(2)
		b.key("a").int32(3)

		for i := 3; i < n; i++ {
			b.key(fmt.Sprint("k", i)).int32(int32(i))
		}

		got := decode(t, b.encode()).Root.Object

		if len(got) != n-1 || got[0].Key != "a" || got[0].Value.Int != 3 || got[1].Key != "b" {
			t.Errorf("with %d members: %+v, want a = 3 first, then b", n, got[:2])
		}
	}
}

func TestDecodeNotKV3(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":              {nil, ErrNotKV3},
		"text":               {[]byte("<!-- kv3 encoding:text -->"), ErrNotKV3},
		"version 0, cut off": {le.AppendUint32(nil, magicLegacy), ErrCorrupt},
		"version 6":          {le.AppendUint32(nil, magicPrefix|6), ErrVersion},
	}

	for name, c := range cases {
		if _, err := Decode(c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: Decode = %v, want %v", name, err, c.want)
		}
	}
}

// Damaged data is an error, never a panic or a read out of bounds.
func TestDecodeDamaged(t *testing.T) {
	for _, v := range variants() {
		b := newBuilder(v.version, v.compression)
		soundEvents(b)

		data := b.encode()

		for n := range data {
			if _, err := Decode(data[:n]); err == nil {
				t.Errorf("%v: Decode of %d of %d bytes succeeded", v, n, len(data))
			}
		}

		for i := range data {
			damaged := bytes.Clone(data)
			damaged[i] ^= 0xFF

			// Most changes are errors, but one to a value is just another
			// value; what matters is that none panics.
			Decode(damaged)
		}
	}
}

func TestDecodeTooDeep(t *testing.T) {
	b := newBuilder(5, compressNone)

	for range maxDepth + 1 {
		b.array(1)
	}

	b.node(nodeNull)

	if _, err := Decode(b.encode()); !errors.Is(err, ErrCorrupt) {
		t.Errorf("Decode of arrays %d deep = %v, want ErrCorrupt", maxDepth+1, err)
	}
}

func FuzzDecode(f *testing.F) {
	for _, v := range variants() {
		b := newBuilder(v.version, v.compression)
		soundEvents(b)
		f.Add(b.encode())
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := Decode(data)
		if err == nil {
			doc.Text()
		}
	})
}

func TestGUIDString(t *testing.T) {
	if got, want := FormatGeneric.String(), "7412167c-06e9-4698-aff2-e63eb59037e7"; got != want {
		t.Errorf("String = %s, want %s", got, want)
	}
}

func TestDecodeVersion0(t *testing.T) {
	for _, encoding := range []GUID{encodingBinary, encodingBinaryLZ4} {
		l := newLegacy()
		l.count(nodeObject, 4)
		l.key("name").string("Generated.Fencer.Hero.Pick.VO", 0)
		l.key("files").count(nodeArray, 2)
		l.string("a.vsnd", 2)
		l.string("b.vsnd", 0)
		l.key("ints").count(nodeArrayTyped, 2)
		l.typ(nodeInt32, 0)
		l.body = le.AppendUint32(l.body, 7)
		l.body = le.AppendUint32(l.body, 8)
		l.key("duration").double(1.5)

		want := obj(
			member("name", str("Generated.Fencer.Hero.Pick.VO")),
			member("files", arr(Value{Type: TypeString, Flag: FlagResourceName, String: "a.vsnd"}, str("b.vsnd"))),
			member("ints", arr(i32(7), i32(8))),
			member("duration", Value{Type: TypeFloat64, Float: 1.5}),
		)

		data := l.encode(encoding)

		if got := decode(t, data).Root; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: Root =\n%s\nwant\n%s", encoding, (&Document{Root: got}).Text(), (&Document{Root: want}).Text())
		}

		for n := range data {
			if _, err := Decode(data[:n]); err == nil {
				t.Errorf("%v: Decode of %d of %d bytes succeeded", encoding, n, len(data))
			}
		}
	}

	// Valve's block compression isn't supported.
	bc := GUID{0x46, 0x1A, 0x79, 0x95, 0xBC, 0x95, 0x6C, 0x4F, 0xA7, 0x0B, 0x05, 0xBC, 0xA1, 0xB7, 0xDF, 0xD2}

	l := newLegacy()
	l.int32(1)

	if _, err := Decode(l.encode(bc)); !errors.Is(err, ErrVersion) {
		t.Errorf("Decode of block compressed data = %v, want ErrVersion", err)
	}
}
