package deploy

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Running reports whether Deadlock is running. It's best effort: when it
// can't tell, treat it as not running, so a check that fails never stops
// mods being applied.
func Running() (bool, error) {
	// A Flatpak's sandbox has processes of its own, so it can't see the
	// game's; the host is asked instead, as Chatterino's Flatpak does to
	// tell whether OBS is running. That needs the manifest's
	// --talk-name=org.freedesktop.Flatpak.
	if _, err := os.Stat("/.flatpak-info"); err == nil {
		return pgrep("flatpak-spawn", "--host", "pgrep")
	}

	return running("/proc")
}

// pgrep runs command (pgrep, or something that runs it) to look for the
// game's process by name, waiting up to a second.
func pgrep(command ...string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := exec.CommandContext(ctx, command[0], append(command[1:], "-xi", exeName)...).Run()

	// pgrep exits with 1 when nothing matches, and more for its own errors.
	var exit *exec.ExitError

	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return false, nil
	default:
		return false, err
	}
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
