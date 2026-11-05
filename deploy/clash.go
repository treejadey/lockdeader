package deploy

import (
	"path"
	"strings"
)

// Clashes says, for each of mods, which of the others change any of the same
// files: the indexes of those in mods, in order. Two that do can't both be
// on, or the one higher in the load order would win every file they share,
// which for mods that overlap only partly leaves a broken mix. Files that
// mods carry for their own sake, which the game doesn't care which copy it
// gets of, don't count (see noise), and nor do files two mods have exactly
// the same copy of, such as the note DeadlockForge puts in every VPK.
func Clashes(mods []PathSet) [][]int {
	// Which mods have each file, so it takes one pass over the paths rather
	// than comparing every mod with every other.
	owners := make(map[string][]int)

	for i, set := range mods {
		for p := range set {
			if !noise(p) {
				owners[p] = append(owners[p], i)
			}
		}
	}

	clashing := make([]map[int]bool, len(mods))

	for p, ids := range owners {
		if len(ids) < 2 {
			continue
		}

		for _, i := range ids {
			for _, j := range ids {
				if i == j || mods[i][p] == mods[j][p] {
					continue
				}

				if clashing[i] == nil {
					clashing[i] = make(map[int]bool)
				}

				clashing[i][j] = true
			}
		}
	}

	result := make([][]int, len(mods))

	for i := range mods {
		for j := range mods {
			if clashing[i][j] {
				result[i] = append(result[i], j)
			}
		}
	}

	return result
}

// noise is true for the files, in lower case, that many mods have whatever
// they change: their own descriptions, and files that tools for making mods
// put in every VPK. Deadlock Mod Manager ignores the same ones.
func noise(p string) bool {
	name := path.Base(p)

	switch {
	case name == "addoninfo.txt", name == "modinfo.json":
		return true
	case strings.HasPrefix(name, "readme"):
		return true
	case p == "panorama/image_compiler.vdata_c":
		return true
	case strings.HasPrefix(p, "materials/default/default_"):
		return true
	}

	return false
}
