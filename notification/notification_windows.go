package notification

import (
	"sync"

	toast "git.sr.ht/~jackmordaunt/go-toast/v2"
)

const appID = "Lockdeader"

var registerOnce sync.Once

// Send may take a moment, so call it from a goroutine.
func Send(title, body string) error {
	// Windows only shows notifications from apps it knows about, so this adds
	// the app under the user's registry key. The GUID is the app's own, for
	// Windows to call back when a notification is clicked.
	var err error

	registerOnce.Do(func() {
		err = toast.SetAppData(toast.AppData{AppID: appID, GUID: "{7BCEF620-2FBD-4063-8980-7993B35385C2}"})
	})

	if err != nil {
		return err
	}

	n := toast.Notification{AppID: appID, Title: title, Body: body}

	return n.Push()
}
