// Package kv3 reads KeyValues3, the format of most of Source 2's data, as
// Source 2's resource compiler writes it into the DATA block of compiled
// files (see package resource): binary, versions 0 to 5, stored as is or
// compressed with LZ4 or Zstandard. It can also write a document out as
// KV3 text, the way Source2Viewer does, to look at or compare.
//
// The binary format is undocumented; this follows ValveResourceFormat's
// reader (github.com/ValveResourceFormat/ValveResourceFormat,
// BinaryKV3.cs), which is MIT licensed.
package kv3

import (
	"encoding/binary"
	"fmt"
)

// A Document is a decoded KV3 file.
type Document struct {
	// What the data is laid out as, e.g. FormatGeneric.
	Format GUID
	Root   Value
}

// A GUID names a KV3 encoding or format. It's kept as stored, which is how
// .NET lays out a GUID: the first three groups little-endian.
type GUID [16]byte

// FormatGeneric is the format of most files, such as sound events.
var FormatGeneric = GUID{0x7C, 0x16, 0x12, 0x74, 0xE9, 0x06, 0x98, 0x46, 0xAF, 0xF2, 0xE6, 0x3E, 0xB5, 0x90, 0x37, 0xE7}

func (g GUID) String() string {
	le := binary.LittleEndian

	return fmt.Sprintf("%08x-%04x-%04x-%x-%x",
		le.Uint32(g[0:]), le.Uint16(g[4:]), le.Uint16(g[6:]), g[8:10], g[10:16])
}

// Type is a value's type. Numbers keep the width they were stored with, so
// a document can be written back as it was.
type Type uint8

const (
	TypeNull Type = iota
	TypeBool
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypeString
	TypeBlob
	TypeArray
	TypeObject
)

func (t Type) String() string {
	names := [...]string{"null", "bool", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32",
		"uint64", "float32", "float64", "string", "blob", "array", "object"}

	if int(t) < len(names) {
		return names[t]
	}

	return fmt.Sprintf("Type(%d)", t)
}

// A Flag marks what a value, usually a string, refers to.
type Flag uint8

const (
	NoFlag Flag = iota
	FlagResource
	FlagResourceName
	FlagPanorama
	FlagSoundEvent
	FlagSubclass
	FlagEntityName
	FlagLocalize
)

func (f Flag) String() string {
	names := [...]string{"", "resource", "resource_name", "panorama", "soundevent", "subclass", "entity_name", "localize"}

	if int(f) < len(names) {
		return names[f]
	}

	return fmt.Sprintf("Flag(%d)", f)
}

// A Value is a KV3 value. Which field holds it depends on its Type.
type Value struct {
	Type Type
	Flag Flag

	Bool   bool
	Int    int64   // Int8 to Int64
	Uint   uint64  // Uint8 to Uint64
	Float  float64 // Float32 and Float64
	String string
	Blob   []byte
	Array  []Value
	Object Object
}

// An Object's members are kept in the order they were stored in.
type Object []Member

type Member struct {
	Key   string
	Value Value
}

// Get returns the value of the member with the key given.
func (o Object) Get(key string) (*Value, bool) {
	for i := range o {
		if o[i].Key == key {
			return &o[i].Value, true
		}
	}

	return nil, false
}
