package kv3

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// The text encoding's GUID, which the header names.
const textEncoding = "e21c7f3c-8a33-41c5-9977-a76d3a32aa0d"

// WriteText writes the document as KV3 text, laid out as Source2Viewer
// lays it out, so the two can be compared line by line.
func (doc *Document) WriteText(w io.Writer) error {
	bw := bufio.NewWriter(w)
	t := &textWriter{w: bw}

	format := "generic"
	if doc.Format != FormatGeneric {
		format = "unknown"
	}

	fmt.Fprintf(bw, "<!-- kv3 encoding:text:version{%s} format:%s:version{%s} -->\n", textEncoding, format, doc.Format)

	t.value("", false, doc.Root, false)

	return bw.Flush()
}

// Text returns the document as KV3 text (see WriteText).
func (doc *Document) Text() string {
	var b strings.Builder

	doc.WriteText(&b)

	return b.String()
}

type textWriter struct {
	w      *bufio.Writer
	indent int
}

func (t *textWriter) indentation() {
	for range t.indent {
		t.w.WriteByte('\t')
	}
}

// value writes v: as an object's member when hasKey, as an array's element
// when inArray, or as the root.
func (t *textWriter) value(key string, hasKey bool, v Value, inArray bool) {
	switch v.Type {
	case TypeObject:
		t.object(key, hasKey, v, inArray)
	case TypeArray:
		t.array(key, hasKey, v, inArray)
	default:
		t.indentation()

		if hasKey {
			t.key(key)
		}

		t.flag(v.Flag)
		t.scalar(v, inArray)

		if inArray {
			t.w.WriteByte(',')
		}

		t.w.WriteByte('\n')
	}
}

func (t *textWriter) object(key string, hasKey bool, v Value, inArray bool) {
	t.indentation()

	if hasKey {
		t.key(key)
	}

	t.flag(v.Flag)

	// After "key = " or a flag, the brace goes on the next line.
	if hasKey || v.Flag != NoFlag {
		t.w.WriteByte('\n')
		t.indentation()
	}

	t.w.WriteString("{\n")
	t.indent++

	for _, m := range v.Object {
		t.value(m.Key, true, m.Value, false)
	}

	t.indent--
	t.indentation()
	t.w.WriteByte('}')

	if inArray {
		t.w.WriteByte(',')
	}

	t.w.WriteByte('\n')
}

func (t *textWriter) array(key string, hasKey bool, v Value, inArray bool) {
	allSimple := true

	for _, e := range v.Array {
		allSimple = allSimple && simple(e.Type)
	}

	t.indentation()

	if hasKey {
		t.key(key)
	}

	t.flag(v.Flag)

	switch {
	// A few numbers go on one line.
	case len(v.Array) <= 4 && allSimple:
		t.w.WriteString("[ ")

		for i, e := range v.Array {
			if i > 0 {
				t.w.WriteString(", ")
			}

			t.flag(e.Flag)
			t.scalar(e, true)
		}

		t.w.WriteString(" ]")
	// More go four to a line.
	case allSimple:
		t.openBracket(hasKey, v.Flag)

		for i, e := range v.Array {
			if i%4 == 0 {
				t.indentation()
			}

			t.flag(e.Flag)
			t.scalar(e, true)
			t.w.WriteByte(',')

			if i < len(v.Array)-1 && (i+1)%4 != 0 {
				t.w.WriteByte(' ')
			} else {
				t.w.WriteByte('\n')
			}
		}

		t.closeBracket()
	default:
		t.openBracket(hasKey, v.Flag)

		for _, e := range v.Array {
			t.value("", false, e, true)
		}

		t.closeBracket()
	}

	if inArray {
		t.w.WriteByte(',')
	}

	t.w.WriteByte('\n')
}

func (t *textWriter) openBracket(hasKey bool, flag Flag) {
	if hasKey || flag != NoFlag {
		t.w.WriteByte('\n')
		t.indentation()
	}

	t.w.WriteString("[\n")
	t.indent++
}

func (t *textWriter) closeBracket() {
	t.indent--
	t.indentation()
	t.w.WriteByte(']')
}

func simple(typ Type) bool {
	switch typ {
	case TypeString, TypeBlob, TypeArray, TypeObject:
		return false
	}

	return true
}

func (t *textWriter) key(key string) {
	if identifier(key) {
		t.w.WriteString(key)
	} else {
		t.quoted(key)
	}

	t.w.WriteString(" = ")
}

func identifier(s string) bool {
	if s == "" || s[0] >= '0' && s[0] <= '9' {
		return false
	}

	for _, c := range []byte(s) {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.'
		if !ok {
			return false
		}
	}

	return true
}

func (t *textWriter) flag(f Flag) {
	if f != NoFlag {
		t.w.WriteString(f.String())
		t.w.WriteByte(':')
	}
}

func (t *textWriter) scalar(v Value, inArray bool) {
	switch v.Type {
	case TypeNull:
		t.w.WriteString("null")
	case TypeBool:
		t.w.WriteString(strconv.FormatBool(v.Bool))
	case TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		t.w.WriteString(strconv.FormatInt(v.Int, 10))
	case TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		t.w.WriteString(strconv.FormatUint(v.Uint, 10))
	case TypeFloat32, TypeFloat64:
		t.w.WriteString(formatFloat(v.Float))
	case TypeString:
		t.text(v.String)
	case TypeBlob:
		t.blob(v.Blob, inArray)
	}
}

// formatFloat formats as Valve's tools do: %f, then without the zeros at
// the end, keeping one digit after the point.
func formatFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}

	s := strconv.FormatFloat(f, 'f', 6, 64)
	s = strings.TrimRight(s, "0")

	if strings.HasSuffix(s, ".") {
		s += "0"
	}

	return s
}

func (t *textWriter) text(s string) {
	// A backslash or carriage return at the end would run into the newline
	// before the closing quotes.
	if strings.Contains(s, "\n") && !strings.HasSuffix(s, `\`) && !strings.HasSuffix(s, "\r") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, `"""`, `\"""`)

		t.w.WriteString("\"\"\"\n" + s + "\n\"\"\"")

		return
	}

	t.quoted(s)
}

var quote = strings.NewReplacer("\n", `\n`, "\t", `\t`, `\`, `\\`, `"`, `\"`)

func (t *textWriter) quoted(s string) {
	t.w.WriteByte('"')
	quote.WriteString(t.w, s)
	t.w.WriteByte('"')
}

func (t *textWriter) blob(b []byte, inArray bool) {
	const hex = "0123456789ABCDEF"

	if len(b) <= 32 {
		t.w.WriteString("#[ ")

		for i, c := range b {
			if i > 0 {
				t.w.WriteByte(' ')
			}

			t.w.WriteByte(hex[c>>4])
			t.w.WriteByte(hex[c&15])
		}

		t.w.WriteString(" ]")

		return
	}

	if inArray {
		t.w.WriteString("#[ \n")
	} else {
		t.w.WriteByte('\n')
		t.indentation()
		t.w.WriteString("#[\n")
	}

	t.indent++
	t.indentation()

	for i, c := range b {
		t.w.WriteByte(hex[c>>4])
		t.w.WriteByte(hex[c&15])

		switch {
		case i == len(b)-1:
		case (i+1)%32 == 0:
			t.w.WriteByte('\n')
			t.indentation()
		default:
			t.w.WriteByte(' ')
		}
	}

	t.indent--
	t.w.WriteByte('\n')
	t.indentation()
	t.w.WriteByte(']')
}
