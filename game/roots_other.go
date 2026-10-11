//go:build !linux && !windows

package game

// Deadlock only runs on Windows, and on Linux through Proton.
func steamRoots() []string {
	return nil
}
