package kv3

import (
	"errors"
	"testing"
)

func TestEncodeRoundTrip(t *testing.T) {
	b := newBuilder(5, compressNone)
	events := soundEvents(b)

	small := obj(
		member("i8", Value{Type: TypeInt8, Int: -5}),
		member("u8", Value{Type: TypeUint8, Uint: 250}),
		member("i16", Value{Type: TypeInt16, Int: -30000}),
		member("u16", Value{Type: TypeUint16, Uint: 60000}),
		member("f", Value{Type: TypeFloat32, Float: 0.25}),
		member("zero", Value{Type: TypeFloat64}),
		member("one", Value{Type: TypeInt64, Int: 1}),
		member("off", Value{Type: TypeUint32}),
		member("no", Value{Type: TypeBool}),
		member("empty", str("")),
		member("nested", arr(arr(), obj(), arr(Value{Type: TypeString, Flag: FlagPanorama, String: "x"}))),
	)

	for name, root := range map[string]Value{"sound events": events, "small values": small, "empty": obj()} {
		doc := &Document{Format: FormatGeneric, Root: root}

		data, err := Encode(doc)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		got := decode(t, data)

		if got.Format != FormatGeneric || !equal(got.Root, root) {
			t.Errorf("%s: decoded =\n%s\nwant\n%s", name, got.Text(), doc.Text())
		}
	}
}

// The header follows the game's own uncompressed files: compressed sizes
// of 0, and a "compressed" total that's the whole size.
func TestEncodeHeader(t *testing.T) {
	data, err := Encode(&Document{Format: FormatGeneric, Root: obj(member("a", i32(1)))})
	if err != nil {
		t.Fatal(err)
	}

	field := func(i int) uint32 { return le.Uint32(data[28+4*i:]) }

	if field(5) != field(6) {
		t.Errorf("compressed total %d, want the uncompressed total %d", field(6), field(5))
	}

	if field(12) != 0 || field(14) != 0 {
		t.Errorf("compressed buffer sizes %d and %d, want 0", field(12), field(14))
	}

	if field(11)+field(13) != field(5) || len(data) != 120+int(field(5)) {
		t.Errorf("buffers of %d and %d in %d bytes, want them to fill it", field(11), field(13), len(data))
	}
}

func TestEncodeBlob(t *testing.T) {
	if _, err := Encode(&Document{Root: arr(Value{Type: TypeBlob, Blob: []byte{1}})}); !errors.Is(err, ErrBlob) {
		t.Errorf("Encode of a blob = %v, want ErrBlob", err)
	}
}

// equal is reflect.DeepEqual, except that an empty array or object is the
// same whether it's nil or not.
func equal(a, b Value) bool {
	if a.Type != b.Type || a.Flag != b.Flag || a.Bool != b.Bool || a.Int != b.Int || a.Uint != b.Uint ||
		a.Float != b.Float || a.String != b.String || string(a.Blob) != string(b.Blob) ||
		len(a.Array) != len(b.Array) || len(a.Object) != len(b.Object) {
		return false
	}

	for i := range a.Array {
		if !equal(a.Array[i], b.Array[i]) {
			return false
		}
	}

	for i := range a.Object {
		if a.Object[i].Key != b.Object[i].Key || !equal(a.Object[i].Value, b.Object[i].Value) {
			return false
		}
	}

	return true
}
