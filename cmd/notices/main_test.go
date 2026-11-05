package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The notices for a platform list the modules compiled in for it, and only
// those, along with the vendored files.
func TestNotices(t *testing.T) {
	// notices reads paths relative to the repository's root.
	t.Chdir("../..")

	for _, tc := range []struct {
		goos      string
		want, not []string
	}{
		{
			goos: "linux",
			want: []string{"github.com/godbus/dbus/v5", "github.com/rymdport/portal"},
			not:  []string{"go-toast", "github.com/go-ole/go-ole"},
		},
		{
			goos: "windows",
			want: []string{"git.sr.ht/~jackmordaunt/go-toast/v2", "github.com/go-ole/go-ole"},
			not:  []string{"github.com/godbus/dbus/v5"},
		},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			t.Setenv("GOOS", tc.goos)

			text, err := notices()
			if err != nil {
				t.Fatal(err)
			}

			want := append([]string{
				"Go standard library go1.",
				"github.com/mappu/miqt v",
				"modernc.org/sqlite",
				"--- PATENTS ---",
				"Inter\nhttps://github.com/rsms/inter",
				"SIL OPEN FONT LICENSE",
				"Fluent UI System Icons",
				"miniaudio",
				"stb_vorbis\n",
				"ALTERNATIVE B - Public Domain",
			}, tc.want...)

			for _, w := range want {
				if !strings.Contains(text, w) {
					t.Errorf("notices for %s don't have %q", tc.goos, w)
				}
			}

			// Build tools that are in go.mod but not in the app.
			for _, n := range append([]string{"goversioninfo", "akavel/rsrc", "makefat"}, tc.not...) {
				if strings.Contains(text, n) {
					t.Errorf("notices for %s have %q", tc.goos, n)
				}
			}

			// stb_vorbis's licence comes out of its source without the code
			// around it.
			if strings.Contains(text, "#endif") || strings.Contains(text, "*/\n") {
				t.Error("stb_vorbis's licence has source code with it")
			}
		})
	}
}

func TestLicenceFilesNone(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/README.md", []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := licenceFiles(dir); !errors.Is(err, errNoLicence) {
		t.Errorf("licenceFiles of a module without one: %v, want errNoLicence", err)
	}
}

// The copy of Go's licence is the same as the one Go ships, where GOROOT has
// it (not on Fedora, which moves it).
func TestGoLicenceUpToDate(t *testing.T) {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatal(err)
	}

	goroot := strings.TrimSpace(string(out))
	checked := 0

	for _, name := range []string{"LICENSE", "PATENTS"} {
		shipped, err := os.ReadFile(filepath.Join(goroot, name))
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			t.Fatal(err)
		}

		ours, err := goLicence.ReadFile("golang/" + name)
		if err != nil {
			t.Fatal(err)
		}

		if string(ours) != string(shipped) {
			t.Errorf("golang/%s differs from %s's; copy it again", name, goroot)
		}

		checked++
	}

	if checked == 0 {
		t.Skipf("GOROOT (%s) has no licence files to compare with", goroot)
	}
}
