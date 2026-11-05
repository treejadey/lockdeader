package kv3

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	large := make([]byte, 33)
	for i := range large {
		large[i] = byte(i)
	}

	ints := func(ns ...int64) Value {
		v := arr()
		for _, n := range ns {
			v.Array = append(v.Array, i32(n))
		}

		return v
	}

	float := func(f float64) Value { return Value{Type: TypeFloat32, Float: f} }

	doc := &Document{Format: FormatGeneric, Root: obj(
		member("name", str("x")),
		member("key with space", i32(1)),
		member("floats", arr(float(float64(float32(3.162656))), float(0.5), float(2), Value{Type: TypeFloat64, Float: 1e-7})),
		member("negative", Value{Type: TypeFloat64, Float: -1.25}),
		member("short", ints(1, 2)),
		member("long", ints(1, 2, 3, 4, 5)),
		member("empty", arr()),
		member("files", arr(Value{Type: TypeString, Flag: FlagResourceName, String: "a.vsnd"}, str("b.vsnd"))),
		member("child", Value{Type: TypeObject, Flag: FlagSubclass, Object: Object{member("x", Value{Type: TypeNull})}}),
		member("list", arr(obj(member("a", Value{Type: TypeBool, Bool: true})))),
		member("nested", arr(ints(7), ints(1, 2, 3, 4, 5))),
		member("text", str("line 1\nline 2")),
		member("escaped", str("say \"hi\"\tnow\\")),
		member("small", Value{Type: TypeBlob, Blob: []byte{0x01, 0xAB}}),
		member("large", Value{Type: TypeBlob, Blob: large}),
	)}

	want := `<!-- kv3 encoding:text:version{e21c7f3c-8a33-41c5-9977-a76d3a32aa0d} format:generic:version{7412167c-06e9-4698-aff2-e63eb59037e7} -->
{
	name = "x"
	"key with space" = 1
	floats = [ 3.162656, 0.5, 2.0, 0.0 ]
	negative = -1.25
	short = [ 1, 2 ]
	long =$
	[
		1, 2, 3, 4,
		5,
	]
	empty = [  ]
	files =$
	[
		resource_name:"a.vsnd",
		"b.vsnd",
	]
	child = subclass:
	{
		x = null
	}
	list =$
	[
		{
			a = true
		},
	]
	nested =$
	[
		[ 7 ],
		[
			1, 2, 3, 4,
			5,
		],
	]
	text = """
line 1
line 2
"""
	escaped = "say \"hi\"\tnow\\"
	small = #[ 01 AB ]
	large =$
	#[
		00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F
		20
	]
}
`

	// Lines that end in a space, which editors tend to strip, end in $.
	want = strings.ReplaceAll(want, "$\n", " \n")

	if got := doc.Text(); got != want {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")

		for i := range min(len(gotLines), len(wantLines)) {
			if gotLines[i] != wantLines[i] {
				t.Fatalf("line %d = %q, want %q\n\nall of it:\n%s", i+1, gotLines[i], wantLines[i], got)
			}
		}

		t.Fatalf("Text =\n%s\nwant\n%s", got, want)
	}
}
