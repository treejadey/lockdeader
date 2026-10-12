package i18n

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Update brings a language's file up to date with the template, as
// gettext's msgmerge does, without its guesses at changed text: messages
// are in the template's order, with its comments and places; translations
// of messages that are still used are kept; new ones are untranslated; and
// translations of ones that aren't used any more are kept as obsolete, in
// case they come back.
func Update(po, template *File) *File {
	nplurals := 2
	if forms, err := ParsePluralForms(po.HeaderField("Plural-Forms")); err == nil {
		nplurals = forms.N
	}

	old := map[string]*Message{}
	for _, m := range po.Messages {
		// An obsolete one only fills in for one that's gone.
		if _, ok := old[m.key()]; !ok || !m.Obsolete {
			old[m.key()] = m
		}
	}

	out := &File{HeaderComments: po.HeaderComments, Header: po.Header}
	used := map[string]bool{}

	for _, t := range template.Messages {
		m := &Message{
			Context:   t.Context,
			ID:        t.ID,
			Plural:    t.Plural,
			Extracted: t.Extracted,
			Files:     t.Files,
			Flags:     slices.Clone(t.Flags),
		}

		used[t.key()] = true
		prev, ok := old[t.key()]

		switch {
		case !ok || (prev.Plural == "") != (t.Plural == ""):
			m.Str = make([]string, 1)
			if t.Plural != "" {
				m.Str = make([]string, nplurals)
			}
		default:
			m.Str = prev.Str
			m.Comments = prev.Comments
			m.Previous = prev.Previous

			// The English plural changed, which may change what the
			// translation should say.
			if prev.Fuzzy() || prev.Plural != t.Plural {
				addFlag(m, true, "fuzzy")
			}
		}

		out.Messages = append(out.Messages, m)
	}

	for _, m := range po.Messages {
		if used[m.key()] || (!m.Obsolete && !slices.ContainsFunc(m.Str, func(s string) bool { return s != "" })) {
			continue
		}

		obsolete := *m
		obsolete.Obsolete = true
		obsolete.Files, obsolete.Extracted = nil, nil
		out.Messages = append(out.Messages, &obsolete)
	}

	return out
}

// Placeholders: Qt's (%1, %L1, %n, %Ln), and Printf's in Go's text (%s,
// %d, %[2]s, …). %% is neither.
var placeholder = regexp.MustCompile(`%(?:%|L?[1-9][0-9]?|L?n|(?:\[[0-9]+\])?[-+# 0]*[0-9.]*[a-zA-Z])`)

// placeholders lists the ones in s, sorted, with Printf's argument indexes
// taken out, since a translation may need them to reorder the arguments.
func placeholders(s string, withN bool) []string {
	var found []string

	for _, p := range placeholder.FindAllString(s, -1) {
		if p == "%%" || (!withN && (p == "%n" || p == "%Ln")) {
			continue
		}

		if i := strings.IndexByte(p, ']'); i >= 0 {
			p = "%" + p[i+1:]
		}

		found = append(found, p)
	}

	slices.Sort(found)

	return found
}

// Check finds mistakes in a language's file that would show in the app:
// plural forms it can't use, and translations whose placeholders don't
// match the English, which would leave out or garble what's filled in.
// Untranslated and fuzzy messages aren't used, so aren't checked.
func Check(po *File) []error {
	var errs []error

	forms, err := ParsePluralForms(po.HeaderField("Plural-Forms"))
	if err != nil {
		errs = append(errs, fmt.Errorf("the header's Plural-Forms: %w", err))
	}

	for _, m := range po.Messages {
		if m.Obsolete || m.Fuzzy() {
			continue
		}

		if m.Plural != "" && forms != nil && len(m.Str) != forms.N {
			errs = append(errs, fmt.Errorf("%q has %d plural forms, not the language's %d", m.ID, len(m.Str), forms.N))

			continue
		}

		for i, s := range m.Str {
			if s == "" {
				continue
			}

			english := m.ID
			if m.Plural != "" && (forms == nil || forms.N == 1 || i > 0) {
				english = m.Plural
			}

			// A plural form may leave the number out, e.g. "a minute ago"
			// for the form that's only for 1.
			withN := m.Plural == ""

			if !slices.Equal(placeholders(english, withN), placeholders(s, withN)) {
				errs = append(errs, fmt.Errorf("%q: the translation %q doesn't have the same placeholders", m.ID, s))
			}
		}
	}

	return errs
}
