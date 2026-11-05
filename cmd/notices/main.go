// Command notices writes the licences of the software built into Lockdeader,
// which have to travel with every copy of the app. It's run by each build,
// for the platform being built for (GOOS, GOARCH), so it lists exactly the Go
// modules compiled in, and never has to be kept up to date by hand:
//
//	go run ./cmd/notices -o <app folder>/THIRD-PARTY-NOTICES.txt
//
// Without -o, it writes to standard output.
//
// It must be run from the repository's root. What a build bundles besides
// the binary, like Qt's DLLs on Windows, is up to the build to add.
package main

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// A component is one piece of third-party software, with the text of its
// licence files.
type component struct {
	name     string
	version  string // "" if it isn't tracked here
	url      string
	licences []licence
}

type licence struct {
	file string
	text string
}

// The files the app embeds or compiles in from its own tree. Their versions
// are in the READMEs next to them.
var vendored = []struct {
	name, url, path string
}{
	{"Inter", "https://github.com/rsms/inter", "fonts/Inter/OFL.txt"},
	{"JetBrains Mono", "https://github.com/JetBrains/JetBrainsMono", "fonts/JetBrainsMono/OFL.txt"},
	{"Fluent UI System Icons", "https://github.com/microsoft/fluentui-system-icons", "qml/icons/LICENSE"},
	{"miniaudio", "https://github.com/mackron/miniaudio", "audio/miniaudio/LICENSE"},
}

// Go's own licence files, which aren't always in GOROOT: Fedora, for one,
// moves them to /usr/share/licenses. A test checks this copy against GOROOT's
// wherever GOROOT has them.
//
//go:embed golang/LICENSE golang/PATENTS
var goLicence embed.FS

// licenceName matches the files in a Go module that are part of its licence:
// LICENSE, COPYING, NOTICE and PATENTS, and AUTHORS, which BSD-style licences
// name as the copyright holders.
var licenceName = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|patents|authors)([-._].*)?$`)

func main() {
	out := flag.String("o", "", "the file to write; standard output if empty")
	flag.Parse()

	text, err := notices()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(1)
	}

	if *out == "" {
		os.Stdout.WriteString(text)

		return
	}

	if err := os.WriteFile(*out, []byte(text), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(1)
	}
}

func notices() (string, error) {
	std, err := goStandardLibrary()
	if err != nil {
		return "", err
	}

	modules, err := goModules()
	if err != nil {
		return "", err
	}

	own, err := vendoredComponents()
	if err != nil {
		return "", err
	}

	components := append(append([]component{std}, modules...), own...)

	var b strings.Builder

	b.WriteString("Third-party software in Lockdeader\n")
	b.WriteString("==================================\n\n")
	b.WriteString("Lockdeader is free software, under the GNU General Public License,\n")
	b.WriteString("version 3 or later (see the LICENSE file next to this one). It's built\n")
	b.WriteString("with the software below, each under its own licence, whose text\n")
	b.WriteString("follows its name.\n")

	for _, c := range components {
		b.WriteString("\n" + strings.Repeat("=", 78) + "\n")
		b.WriteString(c.name)

		if c.version != "" {
			b.WriteString(" " + c.version)
		}

		b.WriteString("\n" + c.url + "\n")

		for _, l := range c.licences {
			fmt.Fprintf(&b, "\n--- %s ---\n\n%s\n", l.file, strings.TrimSpace(l.text))
		}
	}

	return b.String(), nil
}

func goStandardLibrary() (component, error) {
	version, err := goCommand("env", "GOVERSION")
	if err != nil {
		return component{}, err
	}

	var licences []licence

	for _, name := range []string{"LICENSE", "PATENTS"} {
		data, err := goLicence.ReadFile("golang/" + name)
		if err != nil {
			return component{}, err
		}

		licences = append(licences, licence{file: name, text: string(data)})
	}

	return component{name: "Go standard library", version: strings.TrimSpace(version), url: "https://go.dev",
		licences: licences}, nil
}

// goModules lists the modules compiled into the app for the platform being
// built for, in order of their path. Modules only the tests or the build
// tools use aren't in the binary, so they aren't listed.
func goModules() ([]component, error) {
	out, err := goCommand("list", "-deps",
		"-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}", ".")
	if err != nil {
		return nil, err
	}

	var modules []component
	seen := make(map[string]bool)

	for _, line := range strings.Split(out, "\n") {
		if line == "" || seen[line] {
			continue
		}

		seen[line] = true

		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("go list said %q", line)
		}

		path, version, dir := fields[0], fields[1], fields[2]

		licences, err := licenceFiles(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		modules = append(modules, component{name: path, version: version, url: "https://pkg.go.dev/" + path, licences: licences})
	}

	slices.SortFunc(modules, func(a, b component) int { return strings.Compare(a.name, b.name) })

	return modules, nil
}

// goCommand runs go with cgo on, as the app is always built: which files a
// package has, and so what it imports, can depend on it.
func goCommand(args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}

	return string(out), nil
}

var errNoLicence = errors.New("no licence file")

// licenceFiles reads the licence files in dir, by name. It fails if there
// are none, so a new dependency without one can't slip in unnoticed.
func licenceFiles(dir string) ([]licence, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var licences []licence

	for _, e := range entries {
		if e.IsDir() || !licenceName.MatchString(e.Name()) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}

		licences = append(licences, licence{file: e.Name(), text: string(data)})
	}

	if len(licences) == 0 {
		return nil, fmt.Errorf("%w in %s", errNoLicence, dir)
	}

	return licences, nil
}

func vendoredComponents() ([]component, error) {
	var components []component

	for _, v := range vendored {
		data, err := os.ReadFile(filepath.FromSlash(v.path))
		if err != nil {
			return nil, err
		}

		components = append(components, component{name: v.name, url: v.url,
			licences: []licence{{file: filepath.Base(v.path), text: string(data)}}})
	}

	vorbis, err := stbVorbis()
	if err != nil {
		return nil, err
	}

	return append(components, vorbis), nil
}

// stbVorbis takes stb_vorbis's licence from the end of its source file,
// where it is, the only place.
func stbVorbis() (component, error) {
	const path = "audio/miniaudio/stb_vorbis.c"

	data, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		return component{}, err
	}

	text := string(data)

	start := strings.LastIndex(text, "This software is available under 2 licenses")
	if start < 0 {
		return component{}, fmt.Errorf("%s: %w", path, errNoLicence)
	}

	text = strings.TrimSuffix(strings.TrimSpace(text[start:]), "*/")

	return component{name: "stb_vorbis", url: "https://github.com/nothings/stb",
		licences: []licence{{file: "stb_vorbis.c", text: text}}}, nil
}
