package filedialog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rymdport/portal/filechooser"
)

// ChooseFolder asks the user for a folder, starting in start if it's not "".
// It returns "" if they cancel. It blocks until they're done, so call it from
// a goroutine.
func ChooseFolder(parent Window, title, start string) (string, error) {
	handle := ""

	if parent.X11 && parent.ID != 0 {
		handle = fmt.Sprintf("x11:%x", parent.ID)
	}

	uris, err := filechooser.OpenFile(handle, title, &filechooser.OpenFileOptions{
		Directory:     true,
		CurrentFolder: start,
	})
	if err == nil {
		if len(uris) == 0 {
			return "", nil
		}

		// The portal gives a file URI, already unescaped.
		return strings.TrimPrefix(uris[0], "file://"), nil
	}

	// No portal, e.g. on a desktop without xdg-desktop-portal: zenity or
	// kdialog may be there instead.
	folder, fallbackErr := chooseWithZenity(parent, title, start)
	if fallbackErr != nil {
		return "", errors.Join(err, fallbackErr)
	}

	return folder, nil
}
