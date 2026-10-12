// Command translations keeps Lockdeader's translation files up to date. Run
// it from anywhere in the repository:
//
//	go run ./cmd/translations extract
//	go run ./cmd/translations check
//
// extract finds the text marked for translation (qsTr in QML, tr and trN in
// Go), writes it to po/lockdeader.pot, and brings each language's
// po/<language>.po up to date with it. check reports what would go wrong:
// a template that's out of date, and mistakes in the languages' files.
//
// It never translates anything: translators fill in the .po files.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/treejadey/lockdeader/i18n"
)

const usage = `Usage: go run ./cmd/translations <command>

Commands:
  extract  write po/lockdeader.pot from the source, and update each po/*.po
  check    report an out-of-date template, and mistakes in po/*.po
`

func main() {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	root, err := findRoot()
	if err != nil {
		fail(err)
	}

	switch os.Args[1] {
	case "extract":
		err = extract(root)
	case "check":
		err = check(root)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// findRoot finds the repository's root, the folder with qml/Main.qml, from
// the working folder or one above it.
func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "qml", "Main.qml")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("run this inside Lockdeader's repository")
		}

		dir = parent
	}
}

func extract(root string) error {
	template, err := i18n.Extract(os.DirFS(root))
	if err != nil {
		return err
	}

	poDir := filepath.Join(root, "po")

	if err := os.MkdirAll(poDir, 0o755); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(poDir, i18n.TemplateName), template); err != nil {
		return err
	}

	fmt.Printf("%s: %d messages\n", i18n.TemplateName, len(template.Messages))

	languages, err := i18n.FindLanguages(os.DirFS(poDir))
	if err != nil {
		return err
	}

	for _, lang := range languages {
		path := filepath.Join(poDir, lang+".po")

		po, err := readFile(path)
		if err != nil {
			return err
		}

		updated := i18n.Update(po, template)

		if err := writeFile(path, updated); err != nil {
			return err
		}

		fmt.Printf("%s.po: %s\n", lang, progress(updated))
	}

	return nil
}

func check(root string) error {
	poDir := filepath.Join(root, "po")
	ok := true

	template, err := i18n.Extract(os.DirFS(root))
	if err != nil {
		return err
	}

	current, err := os.ReadFile(filepath.Join(poDir, i18n.TemplateName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	var want bytes.Buffer

	if err := template.Write(&want); err != nil {
		return err
	}

	if !bytes.Equal(current, want.Bytes()) {
		fmt.Printf("%s is out of date: run go run ./cmd/translations extract\n", i18n.TemplateName)

		ok = false
	}

	languages, err := i18n.FindLanguages(os.DirFS(poDir))
	if err != nil {
		return err
	}

	for _, lang := range languages {
		po, err := readFile(filepath.Join(poDir, lang+".po"))
		if err != nil {
			fmt.Println(err)

			ok = false

			continue
		}

		fmt.Printf("%s.po: %s\n", lang, progress(po))

		for _, err := range i18n.Check(po) {
			fmt.Printf("%s.po: %v\n", lang, err)

			ok = false
		}
	}

	if !ok {
		return errors.New("found problems")
	}

	return nil
}

func progress(po *i18n.File) string {
	used, translated, fuzzy := 0, 0, 0

	for _, m := range po.Messages {
		if m.Obsolete {
			continue
		}

		used++

		switch {
		case m.Fuzzy():
			fuzzy++
		case m.Translated():
			translated++
		}
	}

	s := fmt.Sprintf("%d of %d translated", translated, used)
	if fuzzy > 0 {
		s += fmt.Sprintf(", %d to check (fuzzy)", fuzzy)
	}

	return s
}

func readFile(path string) (*i18n.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	po, err := i18n.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return po, nil
}

func writeFile(path string, f *i18n.File) error {
	var b bytes.Buffer

	if err := f.Write(&b); err != nil {
		return err
	}

	return os.WriteFile(path, b.Bytes(), 0o644)
}
