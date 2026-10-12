package i18n

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
)

// TemplateName is the template's name in the translations folder, next to
// a <language>.po for each language, e.g. de.po or pt_BR.po.
const TemplateName = "lockdeader.pot"

// A Translator answers Qt's translate calls, in one language or in English.
// It's safe to use from any goroutine.
type Translator struct {
	// The language's file name without .po, e.g. "pt_BR"; "" for English.
	Language string

	// English plurals by message, from the template, which says which
	// messages have them: English needs them too, with no file of its own.
	plurals map[string]string

	messages map[string][]string // translated, by key
	forms    *PluralForms
}

// Load makes a Translator for the first of languages (as the system lists
// them, e.g. "pt-BR", "de") that fsys has a file for, trying "pt_BR" then
// "pt" for "pt-BR". It stops at English, which the app is written in.
//
// It always returns a Translator: when a language's file can't be read, the
// error says why, and the Translator is English.
func Load(fsys fs.FS, languages []string) (*Translator, error) {
	t := &Translator{plurals: map[string]string{}}

	template, err := readFile(fsys, TemplateName)
	if err != nil {
		return t, err
	}

	for _, m := range template.Messages {
		if m.Plural != "" {
			t.plurals[m.key()] = m.Plural
		}
	}

	available, err := FindLanguages(fsys)
	if err != nil {
		return t, err
	}

	lang := Match(available, languages)
	if lang == "" {
		return t, nil
	}

	po, err := readFile(fsys, lang+".po")
	if err != nil {
		return t, err
	}

	forms, err := ParsePluralForms(po.HeaderField("Plural-Forms"))
	if err != nil {
		return t, fmt.Errorf("%s.po: %w", lang, err)
	}

	t.Language = lang
	t.forms = forms
	t.messages = map[string][]string{}

	for _, m := range po.Messages {
		if !m.Obsolete && !m.Fuzzy() && m.Translated() && (m.Plural == "" || len(m.Str) == forms.N) {
			t.messages[m.key()] = m.Str
		}
	}

	return t, nil
}

func readFile(fsys fs.FS, name string) (*File, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}

	f, err := Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	return f, nil
}

// Match returns the first of wanted (BCP 47 or gettext names) that's
// available (gettext names), or "" when English comes first or none are.
func Match(available, wanted []string) string {
	byName := map[string]string{}
	for _, name := range available {
		byName[normalize(name)] = name
	}

	for _, w := range wanted {
		name := normalize(w)

		// Less and less specific: zh_hant_tw, zh_hant, zh.
		for {
			if found, ok := byName[name]; ok {
				return found
			}

			if name == "en" {
				return ""
			}

			i := strings.LastIndexByte(name, '_')
			if i < 0 {
				break
			}

			name = name[:i]
		}
	}

	return ""
}

func normalize(name string) string {
	// A gettext name's encoding and variant, as in sr_RS.UTF-8@latin, don't
	// say anything a file name here would.
	name, _, _ = strings.Cut(name, ".")
	name, _, _ = strings.Cut(name, "@")

	return strings.ToLower(strings.ReplaceAll(name, "-", "_"))
}

// Translate is QTranslator::translate. Qt passes qsTr's disambiguation, and
// n, which is -1 unless the text depends on a number; Qt fills in %n
// afterwards. For a message with a plural, the disambiguation is the
// English plural (see Extract). The context, the QML file's name, isn't
// needed: the same text means the same wherever it is, as in gettext.
func (t *Translator) Translate(source, disambiguation string, n int) string {
	if n >= 0 && disambiguation != "" && t.plurals["\x04"+source] == disambiguation {
		if strs, ok := t.messages["\x04"+source]; ok {
			return strs[t.forms.Index(n)]
		}

		if n == 1 {
			return source
		}

		return disambiguation
	}

	if strs, ok := t.messages[disambiguation+"\x04"+source]; ok {
		return strs[0]
	}

	return source
}

// LogValue describes the Translator for the log.
func (t *Translator) LogValue() slog.Value {
	if t.Language == "" {
		return slog.StringValue("English")
	}

	return slog.StringValue(fmt.Sprintf("%s, %d messages translated", t.Language, len(t.messages)))
}

// FindLanguages lists the languages there's a file for in fsys.
func FindLanguages(fsys fs.FS) ([]string, error) {
	names, err := fs.Glob(fsys, "*.po")
	if err != nil {
		return nil, err
	}

	for i, name := range names {
		names[i] = strings.TrimSuffix(path.Base(name), ".po")
	}

	return names, nil
}
