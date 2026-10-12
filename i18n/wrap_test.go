package i18n

import (
	"bytes"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWriteWraps(t *testing.T) {
	f := &File{
		Header: "Content-Type: text/plain; charset=UTF-8\n",
		Messages: []*Message{
			{
				Files: []string{"qml/AppDialog.qml", "qml/ImageViewer.qml", "qml/InstallDialog.qml", "qml/ModConfigDialog.qml"},
				ID:    "A mod from another tool was found in the game’s addons folder. Lockdeader can take it over.",
				Str:   []string{""},
			},
			{
				ID:  "Two lines,\nthe second ending\n",
				Str: []string{"Fits on one line"},
			},
		},
	}

	want := `msgid ""
msgstr "Content-Type: text/plain; charset=UTF-8\n"

#: qml/AppDialog.qml qml/ImageViewer.qml qml/InstallDialog.qml
#: qml/ModConfigDialog.qml
msgid ""
"A mod from another tool was found in the game’s addons folder. Lockdeader "
"can take it over."
msgstr ""

msgid ""
"Two lines,\n"
"the second ending\n"
msgstr "Fits on one line"
`

	var b bytes.Buffer
	if err := f.Write(&b); err != nil {
		t.Fatal(err)
	}

	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}

// Where gettext's own tools are installed, checks that a file written here
// is laid out as they'd lay it out, so files translators save with them
// round-trip unchanged.
func TestWriteMatchesMsgcat(t *testing.T) {
	msgcat, err := exec.LookPath("msgcat")
	if err != nil {
		t.Skip("msgcat isn't installed")
	}

	t.Run("generated", func(t *testing.T) {
		f := &File{Header: "Content-Type: text/plain; charset=UTF-8\n"}
		rng := rand.New(rand.NewPCG(1, 2))

		for i := range 3000 {
			m := &Message{ID: randomText(rng, i), Str: []string{randomText(rng, -1)}}

			if rng.IntN(3) == 0 {
				m.Context = randomText(rng, -1)
			}

			// Unique, or msgcat drops the repeats.
			for j := range rng.IntN(6) {
				m.Files = append(m.Files, "qml/"+strings.Repeat("x", 1+rng.IntN(40))+string(rune('a'+j))+".qml")
			}

			f.Messages = append(f.Messages, m)
		}

		compareWithMsgcat(t, msgcat, f)
	})

	t.Run("po", func(t *testing.T) {
		paths, err := filepath.Glob("../po/*.po*")
		if err != nil {
			t.Fatal(err)
		}

		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			f, err := Parse(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}

			compareWithMsgcat(t, msgcat, f)
		}
	})
}

func compareWithMsgcat(t *testing.T, msgcat string, f *File) {
	t.Helper()

	var ours bytes.Buffer
	if err := f.Write(&ours); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "test.po")
	if err := os.WriteFile(path, ours.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	theirs, err := exec.Command(msgcat, path).Output()
	if err != nil {
		t.Fatalf("msgcat: %v", err)
	}

	ourLines := strings.Split(ours.String(), "\n")
	theirLines := strings.Split(string(theirs), "\n")

	for i := range min(len(ourLines), len(theirLines)) {
		if ourLines[i] != theirLines[i] {
			from := max(0, i-6)
			t.Fatalf("line %d differs from msgcat's\nours:\n%s\nmsgcat's:\n%s", i+1,
				strings.Join(ourLines[from:min(i+6, len(ourLines))], "\n"),
				strings.Join(theirLines[from:min(i+6, len(theirLines))], "\n"))
		}
	}

	if len(ourLines) != len(theirLines) {
		t.Fatalf("%d lines, msgcat wrote %d", len(ourLines), len(theirLines))
	}
}

// randomText makes UI-like text out of words and the punctuation around
// them. A non-negative seq is added to keep msgids unique.
func randomText(rng *rand.Rand, seq int) string {
	words := []string{
		"a", "mod", "Deadlock", "GameBanana", "folder", "can’t", "isn’t", "it’s",
		"Lockdeader", "addons", "gameinfo.gi", "VPK", "%1", "%n", "50%", "$5",
		"мод", "папці", "Перевірте", "файли", "a-b", "well-known", "and/or",
		"wait…", "yes—no", "x86_64", "C++", "e.g.", "123", "4.5", "#3", "@you",
		"supercalifragilisticexpialidocious", strings.Repeat("x", 30),
	}
	wrappers := [][2]string{
		{"", ""}, {"", ""}, {"", ""}, {"", ""}, {"", ","}, {"", "."}, {"", ":"},
		{"", ";"}, {"", "!"}, {"", "?"}, {"(", ")"}, {"[", "]"}, {"“", "”"},
		{"‘", "’"}, {"\"", "\""}, {"«", "»"}, {"", "…"}, {"", " –"}, {"", " —"},
		{"", " !"}, {"", "\n"}, {"", "\t"}, {"", "\\"}, {"", "\u00a0"}, {"", " "},
	}

	var b strings.Builder

	for i := range 1 + rng.IntN(30) {
		if i > 0 {
			b.WriteByte(' ')
		}

		w := wrappers[rng.IntN(len(wrappers))]
		b.WriteString(w[0] + words[rng.IntN(len(words))] + w[1])
	}

	if seq >= 0 {
		b.WriteString(" " + strings.Repeat("z", 1+seq%7) + strconv.Itoa(seq))
	}

	return b.String()
}
