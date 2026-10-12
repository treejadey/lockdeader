package i18n

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func mustParse(t *testing.T, s string) *File {
	t.Helper()

	f, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}

	return f
}

func write(t *testing.T, f *File) string {
	t.Helper()

	var b bytes.Buffer

	if err := f.Write(&b); err != nil {
		t.Fatal(err)
	}

	return b.String()
}

const samplePO = `# German translation.
msgid ""
msgstr ""
"Language: de\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

# A translator's note.
#. %1 is a mod's name.
#: qml/A.qml qml/B.qml
#, qt-format
msgctxt "verb"
msgid "Install %1"
msgstr "%1 installieren"

#: othermods.go
#, fuzzy, qt-plural-format
msgid "%n mod"
msgid_plural "%n mods"
msgstr[0] "%n Mod"
msgstr[1] "%n Mods"

msgid ""
"Two\n"
"lines \"quoted\""
msgstr "Zwei\nZeilen"

#~ msgid "Gone"
#~ msgstr "Weg"
`

func TestParseAndWrite(t *testing.T) {
	f := mustParse(t, samplePO)

	if f.HeaderField("plural-forms") != "nplurals=2; plural=(n != 1);" || f.HeaderComments[0] != "German translation." {
		t.Errorf("header %q, comments %q", f.Header, f.HeaderComments)
	}

	if len(f.Messages) != 4 {
		t.Fatalf("%d messages, want 4", len(f.Messages))
	}

	m := f.Messages[0]
	if m.Context != "verb" || m.ID != "Install %1" || m.Str[0] != "%1 installieren" ||
		fmt.Sprint(m.Files) != "[qml/A.qml qml/B.qml]" || m.Extracted[0] != "%1 is a mod's name." || m.Comments[0] != "A translator's note." {
		t.Errorf("first message %+v", m)
	}

	if m := f.Messages[1]; m.Plural != "%n mods" || len(m.Str) != 2 || !m.Fuzzy() {
		t.Errorf("plural message %+v", m)
	}

	if m := f.Messages[2]; m.ID != "Two\nlines \"quoted\"" || m.Str[0] != "Zwei\nZeilen" {
		t.Errorf("multi-line message %+v", m)
	}

	if m := f.Messages[3]; !m.Obsolete || m.ID != "Gone" || m.Str[0] != "Weg" {
		t.Errorf("obsolete message %+v", m)
	}

	// Writing it back gives the same file, apart from the order of the
	// header's comments and the long string's layout.
	again := mustParse(t, write(t, f))

	if write(t, again) != write(t, f) || fmt.Sprint(again.Messages[2]) != fmt.Sprint(f.Messages[2]) {
		t.Errorf("didn't survive writing:\n%s", write(t, f))
	}
}

func TestParseErrors(t *testing.T) {
	for _, s := range []string{
		`msgid "a"`,
		`msgid "a" msgstr "b"`,
		"msgid \"a\"\nmsgstr[1] \"b\"",
		"msgid \"a\\q\"\nmsgstr \"\"",
		`"orphan"`,
	} {
		if _, err := Parse(strings.NewReader(s)); err == nil {
			t.Errorf("Parse(%q) succeeded", s)
		}
	}
}

func TestPluralForms(t *testing.T) {
	tests := []struct {
		header string
		want   map[int]int
	}{
		{"nplurals=1; plural=0;", map[int]int{0: 0, 1: 0, 5: 0}},
		{"nplurals=2; plural=(n != 1);", map[int]int{0: 1, 1: 0, 2: 1}},
		{"nplurals=2; plural=n > 1;", map[int]int{0: 0, 1: 0, 2: 1}},
		// Polish.
		{"nplurals=3; plural=(n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
			map[int]int{1: 0, 2: 1, 4: 1, 5: 2, 12: 2, 22: 1, 25: 2, 112: 2}},
		// Russian.
		{"nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
			map[int]int{1: 0, 11: 2, 21: 0, 3: 1, 13: 2, 5: 2}},
		// Arabic.
		{"nplurals=6; plural=(n==0 ? 0 : n==1 ? 1 : n==2 ? 2 : n%100>=3 && n%100<=10 ? 3 : n%100>=11 ? 4 : 5);",
			map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 11: 4, 100: 5, 102: 5, 103: 3}},
		// A rule that gives a form the language doesn't have gets the last.
		{"nplurals=2; plural=n;", map[int]int{0: 0, 1: 1, 7: 1}},
	}

	for _, tt := range tests {
		forms, err := ParsePluralForms(tt.header)
		if err != nil {
			t.Errorf("%s: %v", tt.header, err)

			continue
		}

		for n, want := range tt.want {
			if got := forms.Index(n); got != want {
				t.Errorf("%s: Index(%d) = %d, want %d", tt.header, n, got, want)
			}
		}
	}

	for _, bad := range []string{"", "nplurals=INTEGER; plural=EXPRESSION;", "nplurals=2; plural=(n != 1;", "nplurals=2; plural=n ? 1;", "nplurals=2; plural=x;"} {
		if _, err := ParsePluralForms(bad); err == nil {
			t.Errorf("ParsePluralForms(%q) succeeded", bad)
		}
	}
}

func TestExtract(t *testing.T) {
	fsys := fstest.MapFS{
		"qml/Main.qml": {Data: []byte(`import QtQuick

Item {
    // qsTr("in a comment") isn't a call.
    property string a: qsTr("Play With Mods")
    property string b: "qsTr(\"in a string\")"
    //: %1 is a mod's name.
    property string c: qsTr('Install %1').arg(name)
    property string d: qsTr("Long " +
                            "text")
    property string e: qsTr("Open", "verb")
    property string f: qsTr("%n mod", "%n mods", count(items[0], (1)))
    property string g: object.qsTr("not ours")
    property string h: qsTr("Play With Mods")
    /* qsTr("in a block comment") */
    property string i: qsTr("’ \x41\n")
}
`)},
		"qml/Util.js":    {Data: []byte(`function f() { return qsTr("From JavaScript") }`)},
		"qml/icon.svg":   {Data: []byte(`qsTr("not code")`)},
		"main.go":        {Data: []byte("package main\n\nfunc f() {\n\t//: Shown in a toast.\n\t_ = tr(\"Link\" + \" copied\")\n\t_ = trN(\"%n mod\", \"%n mods\", 2)\n\t_ = tr(`Raw`)\n}\n")},
		"main_test.go":   {Data: []byte("package main\n\nvar _ = tr(\"from a test\")\n")},
		"banana/fake.go": {Data: []byte("package banana\n\nvar _ = tr(\"another package\")\n")},
	}

	f, err := Extract(fsys)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, m := range f.Messages {
		got = append(got, fmt.Sprintf("%s|%s|%s|%v|%v|%v", m.Context, m.ID, m.Plural, m.Files, m.Extracted, m.Flags))
	}

	want := []string{
		"|Link copied||[main.go]|[Shown in a toast.]|[]",
		"|%n mod|%n mods|[main.go qml/Main.qml]|[]|[qt-plural-format]",
		"|Raw||[main.go]|[]|[]",
		"|Play With Mods||[qml/Main.qml]|[]|[]",
		"|Install %1||[qml/Main.qml]|[%1 is a mod's name.]|[qt-format]",
		"|Long text||[qml/Main.qml]|[]|[]",
		"verb|Open||[qml/Main.qml]|[]|[]",
		"|’ A\n||[qml/Main.qml]|[]|[]",
		"|From JavaScript||[qml/Util.js]|[]|[]",
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestExtractErrors(t *testing.T) {
	for name, src := range map[string]string{
		"qml/A.qml": `Item { text: qsTr(name) }`,
		"qml/B.qml": `Item { text: qsTr("a" + name) }`,
		"qml/C.qml": `Item { text: qsTr("%n mod", "", n) }`,
		"a.go":      "package main\n\nvar _ = tr(name)\n",
		"b.go":      "package main\n\nvar _ = trN(\"%n mod\", plural, 2)\n",
	} {
		if _, err := Extract(fstest.MapFS{name: {Data: []byte(src)}}); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err = %v, want one that says where", name, err)
		}
	}

	// The same text can't have two plurals.
	_, err := Extract(fstest.MapFS{"qml/A.qml": {Data: []byte(`Item { a: qsTr("%n mod", "%n mods", n); b: qsTr("%n mod", "%n modules", n) }`)}})
	if err == nil {
		t.Error("two plurals for the same text")
	}
}

func TestUpdate(t *testing.T) {
	template := mustParse(t, `msgid ""
msgstr ""

#: a.go
msgid "Kept"
msgstr ""

#: b.go
msgid "New"
msgstr ""

msgid "%n mod"
msgid_plural "%n mods"
msgstr[0] ""
msgstr[1] ""

msgid "%n file"
msgid_plural "%n files now"
msgstr[0] ""
msgstr[1] ""

msgid "Back"
msgstr ""
`)

	po := mustParse(t, `msgid ""
msgstr "Plural-Forms: nplurals=3; plural=(n==1 ? 0 : n<5 ? 1 : 2);\n"

# Mine.
#: old.go
msgid "Kept"
msgstr "Behalten"

msgid "Removed"
msgstr "Entfernt"

msgid "Untranslated and removed"
msgstr ""

msgid "%n file"
msgid_plural "%n files"
msgstr[0] "a"
msgstr[1] "b"
msgstr[2] "c"

#~ msgid "Back"
#~ msgstr "Zurück"
`)

	got := write(t, Update(po, template))

	want := `msgid ""
msgstr "Plural-Forms: nplurals=3; plural=(n==1 ? 0 : n<5 ? 1 : 2);\n"

# Mine.
#: a.go
msgid "Kept"
msgstr "Behalten"

#: b.go
msgid "New"
msgstr ""

msgid "%n mod"
msgid_plural "%n mods"
msgstr[0] ""
msgstr[1] ""
msgstr[2] ""

#, fuzzy
msgid "%n file"
msgid_plural "%n files now"
msgstr[0] "a"
msgstr[1] "b"
msgstr[2] "c"

msgid "Back"
msgstr "Zurück"

#~ msgid "Removed"
#~ msgstr "Entfernt"
`

	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestCheck(t *testing.T) {
	po := mustParse(t, `msgid ""
msgstr ""
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

msgid "Install %1"
msgstr "Installiere %2"

msgid "%s and %d"
msgstr "%[2]d und %[1]s"

msgid "%n mod; %d left"
msgid_plural "%n mods; %d left"
msgstr[0] "ein Mod; %d übrig"
msgstr[1] "%n Mods; %d übrig"

msgid "%n file"
msgid_plural "%n files"
msgstr[0] "a"

#, fuzzy
msgid "Fuzzy %1"
msgstr "unchecked"

msgid "Untranslated %1"
msgstr ""
`)

	errs := Check(po)

	if len(errs) != 2 || !strings.Contains(errs[0].Error(), "Install %1") || !strings.Contains(errs[1].Error(), "%n file") {
		t.Errorf("Check = %v, want the wrong placeholder and the missing plural form", errs)
	}

	if errs := Check(mustParse(t, "msgid \"\"\nmsgstr \"\"\n")); len(errs) != 1 {
		t.Errorf("Check without Plural-Forms = %v", errs)
	}
}

func TestMatch(t *testing.T) {
	available := []string{"de", "pt_BR", "pt", "zh_Hant", "sr@latin"}

	tests := []struct {
		wanted []string
		want   string
	}{
		{[]string{"de-DE", "de"}, "de"},
		{[]string{"pt-BR"}, "pt_BR"},
		{[]string{"pt-PT"}, "pt"},
		{[]string{"zh-Hant-TW"}, "zh_Hant"},
		{[]string{"fr-FR", "de"}, "de"},
		// English comes before German, so it's English.
		{[]string{"en-GB", "de"}, ""},
		{[]string{"ja"}, ""},
		{[]string{"de_DE.UTF-8"}, "de"},
		{nil, ""},
	}

	for _, tt := range tests {
		if got := Match(available, tt.wanted); got != tt.want {
			t.Errorf("Match(%q) = %q, want %q", tt.wanted, got, tt.want)
		}
	}
}

func TestTranslator(t *testing.T) {
	fsys := fstest.MapFS{
		TemplateName: {Data: []byte(`msgid ""
msgstr ""

msgid "Open"
msgstr ""

msgctxt "verb"
msgid "Open"
msgstr ""

msgid "%n mod"
msgid_plural "%n mods"
msgstr[0] ""
msgstr[1] ""

msgid "%n file"
msgid_plural "%n files"
msgstr[0] ""
msgstr[1] ""
`)},
		"de.po": {Data: []byte(`msgid ""
msgstr ""
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

msgid "Open"
msgstr "Offen"

msgctxt "verb"
msgid "Open"
msgstr "Öffnen"

msgid "%n mod"
msgid_plural "%n mods"
msgstr[0] "%n Mod"
msgstr[1] "%n Mods"

#, fuzzy
msgid "%n file"
msgid_plural "%n files"
msgstr[0] "%n Datei"
msgstr[1] "%n Dateien"
`)},
		"broken.po": {Data: []byte("msgid \"\"\nmsgstr \"\"\n")},
	}

	english, err := Load(fsys, []string{"en-US"})
	if err != nil || english.Language != "" {
		t.Fatalf("English: %v, %q", err, english.Language)
	}

	german, err := Load(fsys, []string{"de-AT"})
	if err != nil || german.Language != "de" {
		t.Fatalf("German: %v, %q", err, german.Language)
	}

	tests := []struct {
		t                      *Translator
		source, disambiguation string
		n                      int
		want                   string
	}{
		{english, "Open", "", -1, "Open"},
		{english, "%n mod", "%n mods", 1, "%n mod"},
		{english, "%n mod", "%n mods", 2, "%n mods"},
		{german, "Open", "", -1, "Offen"},
		{german, "Open", "verb", -1, "Öffnen"},
		{german, "%n mod", "%n mods", 1, "%n Mod"},
		{german, "%n mod", "%n mods", 0, "%n Mods"},
		// Fuzzy isn't used.
		{german, "%n file", "%n files", 3, "%n files"},
		// Qt's own text, which isn't ours.
		{german, "Copy", "", -1, "Copy"},
		{german, "%n byte(s)", "context", 3, "%n byte(s)"},
	}

	for _, tt := range tests {
		if got := tt.t.Translate(tt.source, tt.disambiguation, tt.n); got != tt.want {
			t.Errorf("%s: Translate(%q, %q, %d) = %q, want %q", tt.t.LogValue(), tt.source, tt.disambiguation, tt.n, got, tt.want)
		}
	}

	// A file with no usable plural forms leaves the app in English.
	broken, err := Load(fsys, []string{"broken"})
	if err == nil || broken.Language != "" || broken.Translate("Open", "", -1) != "Open" {
		t.Errorf("broken file: err %v, language %q", err, broken.Language)
	}
}
