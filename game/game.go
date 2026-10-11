// Package game finds where Deadlock is installed, and checks that a folder
// the user picks really is it.
package game

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Deadlock's Steam app id.
const AppID = "1422450"

// ErrNotDeadlock is returned for a folder that isn't Deadlock's, or inside it.
var ErrNotDeadlock = errors.New("game: not a Deadlock folder")

// gameInfo is what makes a folder Deadlock's: the file mods are added to.
var gameInfo = filepath.Join("game", "citadel", "gameinfo.gi")

// Check returns the Deadlock folder dir is, or is inside, e.g. its
// game/citadel folder.
func Check(dir string) (string, error) {
	dir = filepath.Clean(dir)

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", ErrNotDeadlock
	}

	// game/citadel is the deepest folder that has gameinfo.gi in it.
	for range 3 {
		if isDeadlock(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", ErrNotDeadlock
}

func isDeadlock(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, gameInfo))

	return err == nil && info.Mode().IsRegular()
}

// Find looks for Deadlock in the Steam libraries on this computer. It returns
// "" if it isn't installed in any of them.
func Find() string {
	for _, root := range steamRoots() {
		if dir := findIn(root); dir != "" {
			return dir
		}
	}

	return ""
}

// findIn looks through the libraries of the Steam installed at root.
func findIn(root string) string {
	libraries := []string{root}

	if data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf")); err == nil {
		libraries = append(libraries, vdfValues(data, "path")...)
	}

	for _, library := range libraries {
		steamapps := filepath.Join(library, "steamapps")
		installDir := "Deadlock"

		// The manifest says which folder it's in, in case it isn't the usual one.
		if data, err := os.ReadFile(filepath.Join(steamapps, "appmanifest_"+AppID+".acf")); err == nil {
			if values := vdfValues(data, "installdir"); len(values) > 0 && values[0] != "" {
				installDir = values[0]
			}
		}

		dir := filepath.Join(steamapps, "common", installDir)
		if isDeadlock(dir) {
			return dir
		}
	}

	return ""
}

// vdfValues returns every value given for key in a Valve KeyValues file, such
// as libraryfolders.vdf, wherever it is in the file.
func vdfValues(data []byte, key string) []string {
	pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s+"((?:[^"\\]|\\.)*)"`)

	var values []string

	for _, match := range pattern.FindAllSubmatch(data, -1) {
		// Backslashes are escaped, as in Windows paths: C:\\Program Files.
		values = append(values, strings.ReplaceAll(string(match[1]), `\\`, `\`))
	}

	return values
}
