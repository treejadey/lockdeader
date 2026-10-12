// Package i18n reads and writes gettext PO files, extracts the text to
// translate from Lockdeader's QML and Go, and translates with a PO file at
// run time. It has no Qt dependency: the app hands Qt's translate calls to a
// Translator.
package i18n

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// A Message is one entry of a PO file.
type Message struct {
	Context string // msgctxt, from qsTr's disambiguation
	ID      string
	// The English plural, for a message that depends on a number; "" for
	// one that doesn't.
	Plural string
	// The translation, or one per plural form. Empty strings are
	// untranslated.
	Str []string

	Comments  []string // the translator's own ("# ")
	Extracted []string // for translators, from the source ("#.")
	Files     []string // where it's used ("#:")
	Flags     []string // e.g. fuzzy, qt-format ("#,")
	Previous  []string // what msgmerge says it was before ("#|"), kept as is
	Obsolete  bool     // no longer used, kept in case it comes back ("#~")
}

func (m *Message) key() string {
	return m.Context + "\x04" + m.ID
}

// Fuzzy reports whether the translation still needs checking, so isn't used.
func (m *Message) Fuzzy() bool {
	return slices.Contains(m.Flags, "fuzzy")
}

// Translated reports whether every form is translated.
func (m *Message) Translated() bool {
	if len(m.Str) == 0 {
		return false
	}

	for _, s := range m.Str {
		if s == "" {
			return false
		}
	}

	return true
}

// A File is a PO file, or the POT template they're made from.
type File struct {
	// The comments at the top, and the header entry's msgstr.
	HeaderComments []string
	Header         string
	Messages       []*Message
}

// HeaderField returns a header field's value, e.g. "Plural-Forms".
func (f *File) HeaderField(name string) string {
	for line := range strings.SplitSeq(f.Header, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), name) {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

// Parse reads a PO or POT file.
func Parse(r io.Reader) (*File, error) {
	p := poParser{f: &File{}}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1<<20)

	for scanner.Scan() {
		p.line++

		if err := p.parseLine(scanner.Text()); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if err := p.finish(); err != nil {
		return nil, fmt.Errorf("line %d: %w", p.line, err)
	}

	return p.f, nil
}

type poParser struct {
	f    *File
	line int

	m *Message
	// What a continuation line ("…") adds to.
	target *string
	// Set once the entry has a msgid, so the next msgctxt or msgid starts
	// another, even without a blank line between them.
	hasID     bool
	hasHeader bool
}

func (p *poParser) parseLine(line string) error {
	line = strings.TrimSpace(line)

	if line == "" {
		return p.finish()
	}

	obsolete := false

	if rest, ok := strings.CutPrefix(line, "#~"); ok {
		if strings.HasPrefix(rest, "|") {
			p.entry().Previous = append(p.entry().Previous, line)

			return nil
		}

		obsolete = true
		line = strings.TrimSpace(rest)
	}

	if strings.HasPrefix(line, "#") {
		// Comments come before the keywords, so one after them starts the
		// next entry.
		if p.hasID {
			if err := p.finish(); err != nil {
				return err
			}
		}

		m := p.entry()

		switch {
		case strings.HasPrefix(line, "#."):
			m.Extracted = append(m.Extracted, strings.TrimSpace(line[2:]))
		case strings.HasPrefix(line, "#:"):
			m.Files = append(m.Files, strings.Fields(line[2:])...)
		case strings.HasPrefix(line, "#,"):
			for flag := range strings.SplitSeq(line[2:], ",") {
				if flag = strings.TrimSpace(flag); flag != "" {
					m.Flags = append(m.Flags, flag)
				}
			}
		case strings.HasPrefix(line, "#|"):
			m.Previous = append(m.Previous, line)
		default:
			m.Comments = append(m.Comments, strings.TrimPrefix(line[1:], " "))
		}

		return nil
	}

	if strings.HasPrefix(line, `"`) {
		if p.target == nil {
			return fmt.Errorf("a string with no keyword before it")
		}

		s, err := unquote(line)
		if err != nil {
			return err
		}

		*p.target += s

		return nil
	}

	keyword, rest, _ := strings.Cut(line, " ")

	s, err := unquote(strings.TrimSpace(rest))
	if err != nil {
		return err
	}

	if (keyword == "msgctxt" || keyword == "msgid") && p.hasID {
		if err := p.finish(); err != nil {
			return err
		}
	}

	m := p.entry()
	m.Obsolete = m.Obsolete || obsolete

	switch {
	case keyword == "msgctxt":
		m.Context = s
		p.target = &m.Context
	case keyword == "msgid":
		m.ID = s
		p.target = &m.ID
		p.hasID = true
	case keyword == "msgid_plural":
		m.Plural = s
		p.target = &m.Plural
	case keyword == "msgstr":
		m.Str = []string{s}
		p.target = &m.Str[0]
	case strings.HasPrefix(keyword, "msgstr[") && strings.HasSuffix(keyword, "]"):
		i, err := strconv.Atoi(keyword[len("msgstr[") : len(keyword)-1])
		if err != nil || i != len(m.Str) {
			return fmt.Errorf("%s out of order", keyword)
		}

		m.Str = append(m.Str, s)
		p.target = &m.Str[i]
	default:
		return fmt.Errorf("unknown keyword %q", keyword)
	}

	return nil
}

func (p *poParser) entry() *Message {
	if p.m == nil {
		p.m = &Message{}
	}

	return p.m
}

func (p *poParser) finish() error {
	m := p.m
	p.m, p.target, p.hasID = nil, nil, false

	if m == nil {
		return nil
	}

	if m.Str == nil {
		// Comments with no entry, e.g. at the end of the file.
		if m.ID == "" && m.Context == "" {
			return nil
		}

		return fmt.Errorf("%q has no msgstr", m.ID)
	}

	if m.ID == "" && m.Context == "" && !m.Obsolete {
		if p.hasHeader {
			return fmt.Errorf("a second header")
		}

		p.hasHeader = true
		p.f.HeaderComments = m.Comments
		p.f.Header = m.Str[0]

		return nil
	}

	p.f.Messages = append(p.f.Messages, m)

	return nil
}

// unquote reads a PO string, which has C's escapes.
func unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("expected a quoted string, got %q", s)
	}

	s = s[1 : len(s)-1]

	if !strings.Contains(s, `\`) {
		return s, nil
	}

	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])

			continue
		}

		i++

		if i == len(s) {
			return "", fmt.Errorf("a string ends in a backslash")
		}

		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case '"', '\\', '\'', '?':
			b.WriteByte(s[i])
		default:
			return "", fmt.Errorf(`unknown escape \%c`, s[i])
		}
	}

	return b.String(), nil
}

// Write writes f as a PO file.
func (f *File) Write(w io.Writer) error {
	var b bytes.Buffer

	for _, c := range f.HeaderComments {
		writeComment(&b, "#", c)
	}

	// gettext's tools always write the header a line at a time.
	b.WriteString("msgid \"\"\nmsgstr \"\"\n")

	for line := range strings.SplitAfterSeq(f.Header, "\n") {
		if line != "" {
			b.WriteString(quote(line) + "\n")
		}
	}

	for _, m := range f.Messages {
		b.WriteByte('\n')

		for _, c := range m.Comments {
			writeComment(&b, "#", c)
		}

		for _, c := range m.Extracted {
			writeComment(&b, "#.", c)
		}

		for _, file := range m.Files {
			b.WriteString("#: " + file + "\n")
		}

		if len(m.Flags) > 0 {
			b.WriteString("#, " + strings.Join(m.Flags, ", ") + "\n")
		}

		for _, line := range m.Previous {
			b.WriteString(line + "\n")
		}

		prefix := ""
		if m.Obsolete {
			prefix = "#~ "
		}

		if m.Context != "" {
			writeString(&b, prefix, "msgctxt", m.Context)
		}

		writeString(&b, prefix, "msgid", m.ID)

		if m.Plural != "" {
			writeString(&b, prefix, "msgid_plural", m.Plural)

			for i, s := range m.Str {
				writeString(&b, prefix, fmt.Sprintf("msgstr[%d]", i), s)
			}

			continue
		}

		str := ""
		if len(m.Str) > 0 {
			str = m.Str[0]
		}

		writeString(&b, prefix, "msgstr", str)
	}

	_, err := w.Write(b.Bytes())

	return err
}

func writeComment(b *bytes.Buffer, marker, text string) {
	if text == "" {
		b.WriteString(marker + "\n")
	} else {
		b.WriteString(marker + " " + text + "\n")
	}
}

// writeString writes a keyword and its string, a line for each line of the
// string when it has several, as gettext's tools do.
func writeString(b *bytes.Buffer, prefix, keyword, s string) {
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= 1 {
		b.WriteString(prefix + keyword + " " + quote(s) + "\n")

		return
	}

	b.WriteString(prefix + keyword + ` ""` + "\n")

	for _, line := range lines {
		b.WriteString(prefix + quote(line) + "\n")
	}
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')

	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}

	b.WriteByte('"')

	return b.String()
}
