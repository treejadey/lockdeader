//go:build !linux && !windows

package game

import "os/exec"

// Deadlock only runs on Windows, and on Linux through Proton.
func steamCommand() []string {
	return nil
}

func detach(cmd *exec.Cmd) {}
