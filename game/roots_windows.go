package game

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// steamRoots are where Steam says it's installed, then where it's installed
// by default.
func steamRoots() []string {
	var roots []string

	for _, key := range []struct {
		root  registry.Key
		path  string
		value string
	}{
		{registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, "InstallPath"},
	} {
		k, err := registry.OpenKey(key.root, key.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		if dir, _, err := k.GetStringValue(key.value); err == nil && dir != "" {
			roots = append(roots, filepath.Clean(dir))
		}

		k.Close()
	}

	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if dir := os.Getenv(env); dir != "" {
			roots = append(roots, filepath.Join(dir, "Steam"))
		}
	}

	return roots
}
