package i18n

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// TemplateHeader is the header of the POT template. Plural-Forms is left
// for each language's file to fill in, as gettext's own templates do.
const TemplateHeader = "Project-Id-Version: Lockdeader\n" +
	"Language: \n" +
	"MIME-Version: 1.0\n" +
	"Content-Type: text/plain; charset=UTF-8\n" +
	"Content-Transfer-Encoding: 8bit\n" +
	"Plural-Forms: nplurals=INTEGER; plural=EXPRESSION;\n"

// A found message, before ones used in several places are merged.
type found struct {
	file     string
	line     int
	context  string
	id       string
	plural   string
	comments []string
	isQML    bool
}

func (f found) where() string {
	return fmt.Sprintf("%s:%d", f.file, f.line)
}

// Extract makes the template from the text marked for translation in fsys
// (the repository's root): qsTr in qml/*.qml and qml/*.js, and tr and trN
// in the Go files at the root, which are the app's (other packages have no
// Qt). It fails on a call it can't read, e.g. qsTr with a variable, listing
// every one.
//
// A message is the same wherever it's used, unless qsTr's disambiguation
// tells them apart. qsTr with a number takes the English plural in place of
// a disambiguation: qsTr("%n mod", "%n mods", count), as trN does in Go.
func Extract(fsys fs.FS) (*File, error) {
	var all []found
	var errs []error

	qml, err := fs.Glob(fsys, "qml/*")
	if err != nil {
		return nil, err
	}

	goFiles, err := fs.Glob(fsys, "*.go")
	if err != nil {
		return nil, err
	}

	for _, name := range slices.Concat(qml, goFiles) {
		ext := path.Ext(name)
		if ext != ".qml" && ext != ".js" && (ext != ".go" || strings.HasSuffix(name, "_test.go")) {
			continue
		}

		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}

		var msgs []found

		if ext == ".go" {
			msgs, err = extractGo(name, src)
		} else {
			msgs, err = extractQML(name, src)
		}

		all = append(all, msgs...)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return merge(all)
}

// merge makes one message of each that's used in several places, in the
// order they're first found.
func merge(all []found) (*File, error) {
	slices.SortStableFunc(all, func(a, b found) int { return strings.Compare(a.file, b.file) })

	f := &File{
		HeaderComments: []string{
			"The text of Lockdeader's interface, to translate.",
			"Made by `go run ./cmd/translations extract`, so don't edit it by hand.",
		},
		Header: TemplateHeader,
	}

	byKey := map[string]*Message{}
	firstUse := map[string]found{}
	var errs []error

	for _, fm := range all {
		m := &Message{Context: fm.context, ID: fm.id, Plural: fm.plural}
		key := m.key()

		if existing, ok := byKey[key]; ok {
			if existing.Plural != fm.plural {
				errs = append(errs, fmt.Errorf("%s: %q has a different plural than at %s", fm.where(), fm.id, firstUse[key].where()))
			}

			m = existing
		} else {
			if fm.plural != "" {
				m.Str = []string{"", ""}
			} else {
				m.Str = []string{""}
			}

			byKey[key] = m
			firstUse[key] = fm
			f.Messages = append(f.Messages, m)
		}

		if !slices.Contains(m.Files, fm.file) {
			m.Files = append(m.Files, fm.file)
		}

		for _, c := range fm.comments {
			if !slices.Contains(m.Extracted, c) {
				m.Extracted = append(m.Extracted, c)
			}
		}

		// Qt fills in %n for Go too; Go's own placeholders are Printf's.
		addFlag(m, fm.isQML && qtFormat.MatchString(fm.id+fm.plural), "qt-format")
		addFlag(m, fm.plural != "" && strings.Contains(fm.id+fm.plural, "%n"), "qt-plural-format")
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return f, nil
}

var qtFormat = regexp.MustCompile(`%L?[1-9]`)

func addFlag(m *Message, add bool, flag string) {
	if add && !slices.Contains(m.Flags, flag) {
		m.Flags = append(m.Flags, flag)
	}
}

// translatorComments gives a call on line its "//:" comments: the lines
// right above it that are only such a comment, as lupdate reads them.
func translatorComments(notes map[int]string, line int) []string {
	var comments []string

	for l := line - 1; ; l-- {
		note, ok := notes[l]
		if !ok {
			break
		}

		comments = append([]string{note}, comments...)
	}

	if len(comments) == 0 {
		return nil
	}

	return []string{strings.Join(comments, " ")}
}

// extractGo finds tr("…") and trN("…", "…", n) calls. Their text must be a
// string literal, or literals joined with +.
func extractGo(name string, src []byte) ([]found, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	notes := map[int]string{}

	for _, group := range file.Comments {
		for _, c := range group.List {
			if note, ok := strings.CutPrefix(c.Text, "//:"); ok {
				notes[fset.Position(c.Slash).Line] = strings.TrimSpace(note)
			}
		}
	}

	var msgs []found
	var errs []error

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		fn, ok := call.Fun.(*ast.Ident)
		if !ok || (fn.Name != "tr" && fn.Name != "trN") {
			return true
		}

		pos := fset.Position(call.Pos())
		fail := func(what string) {
			errs = append(errs, fmt.Errorf("%s:%d: %s", name, pos.Line, what))
		}

		want := 1
		if fn.Name == "trN" {
			want = 3
		}

		if len(call.Args) != want {
			fail(fmt.Sprintf("%s takes %d arguments", fn.Name, want))

			return true
		}

		m := found{file: name, line: pos.Line, comments: translatorComments(notes, pos.Line)}

		if m.id, ok = goString(call.Args[0]); !ok {
			fail(fn.Name + "'s text must be a string literal, to be found for translating")

			return true
		}

		if fn.Name == "trN" {
			if m.plural, ok = goString(call.Args[1]); !ok || m.plural == "" {
				fail("trN's plural must be a string literal, to be found for translating")

				return true
			}
		}

		msgs = append(msgs, m)

		return true
	})

	return msgs, errors.Join(errs...)
}

func goString(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}

		s, err := strconv.Unquote(e.Value)

		return s, err == nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}

		x, ok := goString(e.X)
		if !ok {
			return "", false
		}

		y, ok := goString(e.Y)

		return x + y, ok
	case *ast.ParenExpr:
		return goString(e.X)
	}

	return "", false
}

// extractQML finds qsTr calls in QML or JavaScript. It skips comments and
// strings, so text that only mentions qsTr isn't taken for a call.
func extractQML(name string, src []byte) ([]found, error) {
	l := &qmlLexer{src: string(src), line: 1, notes: map[int]string{}}
	var msgs []found
	var errs []error

	for l.pos < len(l.src) {
		c := l.src[l.pos]

		switch {
		case c == '\n':
			l.line++
			l.pos++
		case strings.HasPrefix(l.src[l.pos:], "//"):
			l.lineComment()
		case strings.HasPrefix(l.src[l.pos:], "/*"):
			l.blockComment()
		case c == '"' || c == '\'' || c == '`':
			// Strings that aren't qsTr's are only skipped.
			if _, err := l.string(); err != nil {
				errs = append(errs, fmt.Errorf("%s:%d: %w", name, l.line, err))

				return msgs, errors.Join(errs...)
			}
		case isIdentStart(c):
			start := l.pos
			for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
				l.pos++
			}

			// A property or method named qsTr isn't the function.
			if l.src[start:l.pos] != "qsTr" || (start > 0 && l.src[start-1] == '.') {
				continue
			}

			line := l.line

			m, err := l.qsTrCall()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s:%d: %w", name, line, err))

				continue
			}

			if m != nil {
				m.file, m.line, m.isQML = name, line, true
				m.comments = translatorComments(l.notes, line)
				msgs = append(msgs, *m)
			}
		default:
			l.pos++
		}
	}

	return msgs, errors.Join(errs...)
}

type qmlLexer struct {
	src  string
	pos  int
	line int
	// "//:" comments that are alone on their line, by line.
	notes map[int]string
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

func (l *qmlLexer) lineComment() {
	lineStart := strings.LastIndexByte(l.src[:l.pos], '\n') + 1
	alone := strings.TrimSpace(l.src[lineStart:l.pos]) == ""

	end := strings.IndexByte(l.src[l.pos:], '\n')
	if end < 0 {
		end = len(l.src) - l.pos
	}

	text := l.src[l.pos : l.pos+end]

	if note, ok := strings.CutPrefix(text, "//:"); ok && alone {
		l.notes[l.line] = strings.TrimSpace(note)
	}

	l.pos += end
}

func (l *qmlLexer) blockComment() {
	end := strings.Index(l.src[l.pos+2:], "*/")
	if end < 0 {
		end = len(l.src) - l.pos - 2
	}

	l.line += strings.Count(l.src[l.pos:l.pos+2+end], "\n")
	l.pos += 2 + end + 2
	l.pos = min(l.pos, len(l.src))
}

// skipSpace skips whitespace and comments.
func (l *qmlLexer) skipSpace() {
	for l.pos < len(l.src) {
		switch {
		case l.src[l.pos] == '\n':
			l.line++
			l.pos++
		case l.src[l.pos] == ' ' || l.src[l.pos] == '\t' || l.src[l.pos] == '\r':
			l.pos++
		case strings.HasPrefix(l.src[l.pos:], "//"):
			l.lineComment()
		case strings.HasPrefix(l.src[l.pos:], "/*"):
			l.blockComment()
		default:
			return
		}
	}
}

// string reads a string literal, decoding its escapes.
func (l *qmlLexer) string() (string, error) {
	quote := l.src[l.pos]
	l.pos++

	var b strings.Builder

	for l.pos < len(l.src) {
		c := l.src[l.pos]

		switch {
		case c == quote:
			l.pos++

			return b.String(), nil
		case c == '\n' && quote != '`':
			return "", errors.New("a string isn't closed")
		case c == '\\':
			if err := l.escape(&b); err != nil {
				return "", err
			}
		default:
			if c == '\n' {
				l.line++
			}

			b.WriteByte(c)
			l.pos++
		}
	}

	return "", errors.New("a string isn't closed")
}

func (l *qmlLexer) escape(b *strings.Builder) error {
	l.pos++

	if l.pos >= len(l.src) {
		return errors.New("a string ends in a backslash")
	}

	c := l.src[l.pos]
	l.pos++

	switch c {
	case 'n':
		b.WriteByte('\n')
	case 't':
		b.WriteByte('\t')
	case 'r':
		b.WriteByte('\r')
	case 'b':
		b.WriteByte('\b')
	case 'f':
		b.WriteByte('\f')
	case 'v':
		b.WriteByte('\v')
	case '0':
		b.WriteByte(0)
	case '\n':
		// A line continuation.
		l.line++
	case 'x', 'u':
		digits := 2
		if c == 'u' {
			digits = 4
		}

		if c == 'u' && l.pos < len(l.src) && l.src[l.pos] == '{' {
			end := strings.IndexByte(l.src[l.pos:], '}')
			if end < 0 {
				return errors.New(`\u{ isn't closed`)
			}

			r, err := strconv.ParseUint(l.src[l.pos+1:l.pos+end], 16, 32)
			if err != nil || !utf8.ValidRune(rune(r)) {
				return fmt.Errorf(`bad escape \u%s`, l.src[l.pos:l.pos+end+1])
			}

			b.WriteRune(rune(r))
			l.pos += end + 1

			return nil
		}

		if l.pos+digits > len(l.src) {
			return fmt.Errorf(`bad escape \%c`, c)
		}

		r, err := strconv.ParseUint(l.src[l.pos:l.pos+digits], 16, 32)
		if err != nil {
			return fmt.Errorf(`bad escape \%c%s`, c, l.src[l.pos:l.pos+digits])
		}

		b.WriteRune(rune(r))
		l.pos += digits
	default:
		b.WriteByte(c)
	}

	return nil
}

// literal reads a string literal, or literals joined with +, which is all
// lupdate reads too. ok is false, with nothing read, if there isn't one.
func (l *qmlLexer) literal() (s string, ok bool, err error) {
	l.skipSpace()

	if l.pos >= len(l.src) || (l.src[l.pos] != '"' && l.src[l.pos] != '\'') {
		return "", false, nil
	}

	for {
		part, err := l.string()
		if err != nil {
			return "", false, err
		}

		s += part
		l.skipSpace()

		if l.pos >= len(l.src) || l.src[l.pos] != '+' {
			return s, true, nil
		}

		l.pos++
		l.skipSpace()

		if l.pos >= len(l.src) || (l.src[l.pos] != '"' && l.src[l.pos] != '\'') {
			return "", false, errors.New("qsTr's text must be string literals only")
		}
	}
}

// qsTrCall reads a call's arguments, just after "qsTr". It returns nil
// for a mention that isn't a call.
func (l *qmlLexer) qsTrCall() (*found, error) {
	l.skipSpace()

	if l.pos >= len(l.src) || l.src[l.pos] != '(' {
		return nil, nil
	}

	l.pos++

	text, ok, err := l.literal()
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, errors.New("qsTr's text must be a string literal, to be found for translating")
	}

	m := &found{id: text}

	if l.pos < len(l.src) && l.src[l.pos] == ')' {
		l.pos++

		return m, nil
	}

	if l.pos >= len(l.src) || l.src[l.pos] != ',' {
		return nil, errors.New("qsTr's text must be string literals only")
	}

	l.pos++

	second, ok, err := l.literal()
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, errors.New("qsTr's disambiguation or plural must be a string literal")
	}

	if l.pos < len(l.src) && l.src[l.pos] == ')' {
		l.pos++
		m.context = second

		return m, nil
	}

	if l.pos >= len(l.src) || l.src[l.pos] != ',' {
		return nil, errors.New("qsTr's disambiguation or plural must be string literals only")
	}

	if second == "" {
		return nil, errors.New(`qsTr with a number takes the English plural as its second argument, e.g. qsTr("%n mod", "%n mods", count)`)
	}

	m.plural = second

	// The number is any expression; the call ends at its closing bracket.
	l.pos++

	for depth := 0; l.pos < len(l.src); {
		c := l.src[l.pos]

		switch {
		case c == '(' || c == '[' || c == '{':
			depth++
			l.pos++
		case c == ')' && depth == 0:
			l.pos++

			return m, nil
		case c == ')' || c == ']' || c == '}':
			depth--
			l.pos++
		case c == '"' || c == '\'' || c == '`':
			if _, err := l.string(); err != nil {
				return nil, err
			}
		case c == '/' || c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if c == '/' && !strings.HasPrefix(l.src[l.pos:], "//") && !strings.HasPrefix(l.src[l.pos:], "/*") {
				l.pos++

				continue
			}

			l.skipSpace()
		default:
			l.pos++
		}
	}

	return nil, errors.New("qsTr's call isn't closed")
}
