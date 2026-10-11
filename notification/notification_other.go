//go:build !linux && !windows

package notification

// Send does nothing yet on other platforms.
func Send(title, body string) error {
	return nil
}
