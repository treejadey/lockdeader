package i18n

import (
	"bytes"
	"strings"
	"unicode"
)

// Lines are laid out as gettext's tools lay them out, so a file a translator
// saved with one of them (Gtranslator, Poedit, msgmerge) is written back
// with nothing changed but what had to.

// gettext's default page width.
const pageWidth = 79

// writeString writes a keyword and its string. The string goes on the
// keyword's line if it fits there and has no line break before its end.
// Otherwise the keyword gets "" and the string follows, a line for each of
// its lines, each wrapped at the page width.
func writeString(b *bytes.Buffer, prefix, keyword, s string) {
	var portions [][]unit

	for line := range strings.SplitAfterSeq(s, "\n") {
		if line != "" {
			portions = append(portions, units(line))
		}
	}

	if len(portions) == 0 {
		b.WriteString(prefix + keyword + " \"\"\n")

		return
	}

	// Columns for a line's text, between its quotes.
	width := pageWidth - 2 - len(prefix)
	// Where the first line's text starts, after `keyword "`.
	start := len(keyword) + 1

	if len(portions) == 1 && len(breaks(portions[0], start, width)) == 0 {
		b.WriteString(prefix + keyword + " " + joinUnits(portions[0]) + "\n")

		return
	}

	b.WriteString(prefix + keyword + " \"\"\n")

	for _, portion := range portions {
		from := 0

		for _, at := range append(breaks(portion, 0, width), len(portion)) {
			b.WriteString(prefix + joinUnits(portion[from:at]) + "\n")
			from = at
		}
	}
}

// writeReferences writes the "#:" lines, starting another when the next
// file would go past the page width.
func writeReferences(b *bytes.Buffer, files []string) {
	if len(files) == 0 {
		return
	}

	b.WriteString("#:")
	column := 2

	for _, file := range files {
		if column > 2 && column+1+len(file) > pageWidth {
			b.WriteString("\n#:")
			column = 2
		}

		b.WriteString(" " + file)
		column += 1 + len(file)
	}

	b.WriteByte('\n')
}

// A unit is one character of a string as it's written: escaped, so `\n` is
// one unit of two columns, which a line never breaks inside.
type unit struct {
	text string
	// How the unit breaks with what's before it and after it, which differ
	// for an escape.
	first, last breakClass
}

func units(s string) []unit {
	var us []unit

	for _, r := range s {
		text := quote(string(r))
		text = text[1 : len(text)-1]

		// gettext finds line breaks in the escaped text, where an escape is
		// a backslash, which counts as a prefix like "$", then a character
		// such as the quote of `\"`.
		u := unit{text, classOf(r), classOf(r)}
		if len(text) == 2 && text[0] == '\\' {
			u.first = classPR
			u.last = classOf(rune(text[1]))
		}

		us = append(us, u)
	}

	return us
}

func joinUnits(us []unit) string {
	var b strings.Builder
	b.WriteByte('"')

	for _, u := range us {
		b.WriteString(u.text)
	}

	b.WriteByte('"')

	return b.String()
}

// breaks returns where to start new lines, as indexes into us, filling each
// line up to width columns; the first starts at column start. Like
// gettext's (libunistring's ulc_width_linebreaks), it breaks only where
// Unicode's line breaking rules allow, and leaves a line too long when
// there's nowhere to break it.
func breaks(us []unit, start, width int) []int {
	var result []int

	// The last place a line could break, and the column there.
	last := -1
	lastColumn := start
	// The width of the text since then.
	piece := 0

	for i, u := range us {
		if i > 0 && canBreak(us, i) {
			if last >= 0 && lastColumn+piece > width {
				result = append(result, last)
				lastColumn = 0
			}

			last = i
			lastColumn += piece
			piece = 0
		}

		piece += len([]rune(u.text))
	}

	if last >= 0 && lastColumn+piece > width {
		result = append(result, last)
	}

	return result
}

// The line breaking classes of Unicode Standard Annex #14 that UI text uses.
type breakClass int

const (
	classAL breakClass = iota // letters, and anything not listed
	classSP                   // space
	classBA                   // break after: en dash, vertical bar
	classB2                   // em dash
	classHY                   // hyphen-minus
	classOP                   // opening punctuation
	classCL                   // closing punctuation
	classEX                   // ! ?
	classIS                   // , . : ;
	classSY                   // /
	classQU                   // quotation marks and apostrophes
	classIN                   // …
	classPR                   // prefixes: $ + \
	classPO                   // postfixes: %
	classNU                   // digits
	classGL                   // no-break spaces
	classCM                   // combining marks
)

func classOf(r rune) breakClass {
	switch r {
	case ' ':
		return classSP
	case '–', '|', '\t':
		return classBA
	case '—':
		return classB2
	case '-':
		return classHY
	case '(', '[', '{', '¡', '¿':
		return classOP
	case ')', ']', '}':
		return classCL
	case '!', '?':
		return classEX
	case ',', '.', ':', ';':
		return classIS
	case '/':
		return classSY
	case '"', '\'', '«', '»', '‘', '’', '“', '”', '‹', '›':
		return classQU
	case '…':
		return classIN
	case '$', '+', '\\':
		return classPR
	case '%':
		return classPO
	case ' ', ' ', ' ':
		return classGL
	}

	switch {
	case unicode.IsDigit(r):
		return classNU
	case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r):
		return classCM
	}

	return classAL
}

// canBreak says whether a line may start at us[i], following the rules of
// UAX #14 that apply to these classes.
func canBreak(us []unit, i int) bool {
	// gettext's own rule: not just before the \n that ends a line.
	if us[i].text == `\n` {
		return false
	}

	next := us[i].first

	// Not before a space or combining mark, or before closing punctuation
	// (LB7, LB9, LB13).
	switch next {
	case classSP, classCM, classCL, classEX, classIS, classSY:
		return false
	}

	// What's before, ignoring spaces.
	j := i - 1
	for j >= 0 && us[j].last == classSP {
		j--
	}

	before := classAL
	if j >= 0 {
		before = us[j].last
	}

	afterSpace := j < i-1

	if afterSpace {
		// LB14, LB17; otherwise LB18 breaks after spaces.
		return before != classOP && !(before == classB2 && next == classB2)
	}

	switch {
	// LB12, LB12a
	case before == classGL || next == classGL:
		return false
	// LB14
	case before == classOP:
		return false
	// LB19
	case before == classQU || next == classQU:
		return false
	// LB21, LB22
	case next == classBA || next == classHY || next == classIN:
		return false
	// LB17
	case before == classB2 && next == classB2:
		return false
	// LB23 to LB25: numbers with letters, prefixes and postfixes
	case before == classNU && (next == classAL || next == classNU || next == classPO || next == classPR):
		return false
	case next == classNU && (before == classAL || before == classPR || before == classPO ||
		before == classHY || before == classOP || before == classIS || before == classSY):
		return false
	case (before == classPR || before == classPO) && next == classAL:
		return false
	case before == classAL && (next == classPR || next == classPO):
		return false
	// LB28, LB29
	case next == classAL && (before == classAL || before == classIS):
		return false
	// LB30
	case before == classCL && (next == classAL || next == classNU):
		return false
	case next == classOP && (before == classAL || before == classNU):
		return false
	}

	// LB31
	return true
}
