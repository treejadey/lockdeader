package game

import (
	"os"
	"path/filepath"
)

// steamRoots are where Steam is usually installed: by the distribution's
// package or Valve's installer, as a Flatpak, or as a Snap.
func steamRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var roots []string

	for _, dir := range []string{
		".local/share/Steam",
		".steam/steam",
		".steam/root",
		".steam/debian-installation",
		".var/app/com.valvesoftware.Steam/.local/share/Steam",
		".var/app/com.valvesoftware.Steam/data/Steam",
		"snap/steam/common/.local/share/Steam",
	} {
		roots = append(roots, filepath.Join(home, dir))
	}

	return roots
}
