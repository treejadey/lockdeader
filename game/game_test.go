package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// install makes dir look like a Deadlock install.
func install(t *testing.T, dir string) {
	t.Helper()

	path := filepath.Join(dir, "game", "citadel", "gameinfo.gi")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("GameInfo {}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, data string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheck(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Deadlock")
	install(t, dir)

	for _, picked := range []string{dir, filepath.Join(dir, "game"), filepath.Join(dir, "game", "citadel")} {
		if got, err := Check(picked); err != nil || got != dir {
			t.Errorf("Check(%s) = %q, %v; want %q", picked, got, err, dir)
		}
	}

	for _, picked := range []string{t.TempDir(), filepath.Dir(dir), filepath.Join(dir, "missing")} {
		if got, err := Check(picked); !errors.Is(err, ErrNotDeadlock) {
			t.Errorf("Check(%s) = %q, %v; want ErrNotDeadlock", picked, got, err)
		}
	}
}

func TestFindInOtherLibrary(t *testing.T) {
	root := t.TempDir()
	library := t.TempDir()

	// The path is escaped as Steam escapes Windows paths.
	escaped := strings.ReplaceAll(library, `\`, `\\`)
	write(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), `"libraryfolders"
{
	"0"
	{
		"path"		"`+strings.ReplaceAll(root, `\`, `\\`)+`"
		"apps" { "220" "5774110510" }
	}
	"1"
	{
		"path"		"`+escaped+`"
		"apps" { "1422450" "40000000000" }
	}
}`)

	// Installed in a folder other than the usual one.
	write(t, filepath.Join(library, "steamapps", "appmanifest_1422450.acf"), `"AppState"
{
	"appid"		"1422450"
	"installdir"		"Deadlock Playtest"
}`)

	want := filepath.Join(library, "steamapps", "common", "Deadlock Playtest")
	install(t, want)

	if got := findIn(root); got != want {
		t.Errorf("findIn() = %q, want %q", got, want)
	}

	if got := findIn(t.TempDir()); got != "" {
		t.Errorf("findIn() with no Steam = %q, want \"\"", got)
	}
}

func TestVDFValues(t *testing.T) {
	data := []byte(`"a" { "path"	"C:\\Program Files (x86)\\Steam" "label" "" } "b" { "path" "D:\\Games" }`)

	got := vdfValues(data, "path")
	want := []string{`C:\Program Files (x86)\Steam`, `D:\Games`}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("vdfValues() = %q, want %q", got, want)
	}
}

func TestSplitOptions(t *testing.T) {
	for options, want := range map[string]string{
		"":                            "[]",
		"   ":                         "[]",
		"-novid -high":                "[-novid -high]",
		`+exec "my config.cfg"  -dev`: "[+exec my config.cfg -dev]",
		`-name 'it''s'`:               "[-name its]",
		`-empty "" -last`:             `[-empty  -last]`,
		"-a\t-b\n-c":                  "[-a -b -c]",
	} {
		if got := fmt.Sprint(SplitOptions(options)); got != want {
			t.Errorf("SplitOptions(%q) = %s, want %s", options, got, want)
		}
	}
}

func TestSteamURL(t *testing.T) {
	if got, want := SteamURL([]string{"-novid", "-high"}), "steam://run/1422450//-novid%20-high/"; got != want {
		t.Errorf("SteamURL() = %q, want %q", got, want)
	}

	if got, want := SteamURL(nil), "steam://run/1422450///"; got != want {
		t.Errorf("SteamURL(nil) = %q, want %q", got, want)
	}
}
