package notification

import "github.com/godbus/dbus/v5"

// Send blocks until the notification server answers, so call it from a
// goroutine.
func Send(title, body string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}

	// Lets the desktop show the app's name and icon, and group its
	// notifications.
	hints := map[string]dbus.Variant{"desktop-entry": dbus.MakeVariant("io.github.treejadey.Lockdeader")}

	// The arguments are the app's name, the id of a notification to replace
	// (none), an icon (the app's), the title and body, actions (none), the
	// hints, and how long to show it for (the server's default).
	call := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").
		Call("org.freedesktop.Notifications.Notify", 0,
			"Lockdeader", uint32(0), "", title, body, []string{}, hints, int32(-1))

	return call.Err
}
