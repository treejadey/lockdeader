package deploy

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Running reports whether Deadlock is running. It's best effort: when it
// can't tell, treat it as not running, so a check that fails never stops
// mods being applied.
func Running() (bool, error) {
	return running("/proc")
}

// running looks through the processes listed in procDir. Deadlock runs
// under Proton here, so it shows up as deadlock.exe, with a Windows path in
// its command line.
func running(procDir string) (bool, error) {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil || !e.IsDir() {
			continue
		}

		dir := filepath.Join(procDir, e.Name())

		// Processes come and go while this runs; one that's gone is skipped.
		if comm, err := os.ReadFile(filepath.Join(dir, "comm")); err == nil {
			if isGame(strings.TrimSpace(string(comm))) {
				return true, nil
			}
		} else if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, fs.ErrPermission) {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline"))
		if err != nil {
			continue
		}

		argv0, _, _ := bytes.Cut(cmdline, []byte{0})
		if isGame(string(argv0)) {
			return true, nil
		}
	}

	return false, nil
}

func isGame(name string) bool {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}

	return strings.EqualFold(name, exeName)
}
