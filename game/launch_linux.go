package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// steamCommand is how to run Steam: from the distribution's package or
// Valve's installer, or as a Flatpak. nil if neither is installed.
func steamCommand() []string {
	if path, err := exec.LookPath("steam"); err == nil {
		return []string{path}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	flatpak, err := exec.LookPath("flatpak")
	if err != nil {
		return nil
	}

	if _, err := os.Stat(filepath.Join(home, ".var", "app", "com.valvesoftware.Steam")); err != nil {
		return nil
	}

	return []string{flatpak, "run", "com.valvesoftware.Steam"}
}

// detach runs Steam in its own session, so it isn't stopped along with the
// app, e.g. by closing the terminal the app was started from.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
