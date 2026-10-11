package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// steamCommand is steam.exe, as Steam's registry key gives it, or where it's
// installed by default. nil if it isn't there.
func steamCommand() []string {
	var candidates []string

	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE); err == nil {
		if exe, _, err := k.GetStringValue("SteamExe"); err == nil && exe != "" {
			candidates = append(candidates, filepath.Clean(exe))
		}

		k.Close()
	}

	for _, root := range steamRoots() {
		candidates = append(candidates, filepath.Join(root, "steam.exe"))
	}

	for _, exe := range candidates {
		if info, err := os.Stat(exe); err == nil && info.Mode().IsRegular() {
			return []string{exe}
		}
	}

	return nil
}

// detach starts Steam without a console window, and apart from the app, so
// closing one doesn't close the other.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
}
