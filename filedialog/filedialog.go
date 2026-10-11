// Package filedialog asks the user for a folder with the system's own
// dialog: through xdg-desktop-portal on Linux, so it matches the desktop,
// and Windows' own folder picker on Windows.
package filedialog

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ncruces/zenity"
)

// Window is the window a dialog is for, so it can open over it: an HWND on
// Windows, or an X11 window. A Wayland window has no id a dialog can use.
type Window struct {
	ID  uintptr
	X11 bool
}

// chooseWithZenity uses the system's dialog on Windows and macOS, and the
// zenity or kdialog commands on Linux.
func chooseWithZenity(parent Window, title, start string) (string, error) {
	options := []zenity.Option{zenity.Title(title), zenity.Directory(), zenity.Modal()}

	// A trailing separator opens the folder itself, rather than its parent
	// with the folder picked.
	if start != "" {
		options = append(options, zenity.Filename(start+string(os.PathSeparator)))
	}

	if parent.ID != 0 && (parent.X11 || filepath.Separator == '\\') {
		options = append(options, zenity.Attach(parent.ID))
	}

	folder, err := zenity.SelectFile(options...)
	if errors.Is(err, zenity.ErrCanceled) {
		return "", nil
	}

	return folder, err
}
