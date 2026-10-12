package main

import (
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/i18n"
)

const testTemplate = `msgid ""
msgstr ""

msgid "Link copied"
msgstr ""

msgid "%n year ago"
msgid_plural "%n years ago"
msgstr[0] ""
msgstr[1] ""
`

// Polish-like plural forms, with three.
const testPO = `msgid ""
msgstr ""
"Plural-Forms: nplurals=3; plural=(n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);\n"

msgid "Link copied"
msgstr "Link kopiert"

msgid "%n year ago"
msgid_plural "%n years ago"
msgstr[0] "one: %n"
msgstr[1] "few: %n"
msgstr[2] "many: %n"
`

// Qt passes Go's and QML's text through to the translator, and fills in %n.
func TestTranslations(t *testing.T) {
	files := fstest.MapFS{
		i18n.TemplateName: {Data: []byte(testTemplate)},
		"xx.po":           {Data: []byte(testPO)},
	}

	english, err := i18n.Load(files, []string{"en-US", "xx"})
	if err != nil {
		t.Fatal(err)
	}

	translated, err := i18n.Load(files, []string{"xx-YY"})
	if err != nil {
		t.Fatal(err)
	}

	mainthread.Wait(func() {
		engine := qml.NewQQmlEngine()
		defer engine.Delete()

		qsTr := func(n int) string {
			return engine.Evaluate(`qsTr("%n year ago", "%n years ago", ` + strconv.Itoa(n) + `)`).ToString()
		}

		// English needs the translator too, for its plurals. Other tests
		// expect English, so it's left in place.
		installTranslator(english)
		defer installTranslator(english)

		if got := trN("%n year ago", "%n years ago", 1); got != "1 year ago" {
			t.Errorf("English trN(1) = %q", got)
		}

		if got := qsTr(5); got != "5 years ago" {
			t.Errorf("English qsTr(5) = %q", got)
		}

		if got := tr("Not in the template"); got != "Not in the template" {
			t.Errorf("English tr = %q", got)
		}

		installTranslator(translated)

		if got := tr("Link copied"); got != "Link kopiert" {
			t.Errorf("tr = %q", got)
		}

		for n, want := range map[int]string{1: "one: 1", 3: "few: 3", 5: "many: 5", 22: "few: 22"} {
			if got := trN("%n year ago", "%n years ago", n); got != want {
				t.Errorf("trN(%d) = %q, want %q", n, got, want)
			}

			if got := qsTr(n); got != want {
				t.Errorf("qsTr(%d) = %q, want %q", n, got, want)
			}
		}
	})
}
