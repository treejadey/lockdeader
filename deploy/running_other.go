//go:build !linux && !windows

package deploy

// Running reports whether Deadlock is running. It's best effort: when it
// can't tell, treat it as not running, so a check that fails never stops
// mods being applied.
func Running() (bool, error) {
	return false, nil
}
