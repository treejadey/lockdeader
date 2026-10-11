package main

import (
	"errors"
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/game"
)

// gameLauncher starts Deadlock through Steam. QML starts it by writing "play":
//
//	"mods":    with the mods that are turned on
//	"vanilla": without any mods
//
// Go adds the launch options from the settings. As in the clipboard bridge,
// "message" says how it went, and "messageId" goes up by one with each.
//
// All methods must be called on the Qt main thread.
type gameLauncher struct {
	*qml.QQmlPropertyMap

	launchOptions func() string

	// game.Launch and opening a URL, other than in tests.
	launch  func(args []string) error
	openURL func(url string) bool

	messages int
}

func newGameLauncher(launchOptions func() string) *gameLauncher {
	l := &gameLauncher{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		launchOptions:   launchOptions,
		launch:          game.Launch,
		openURL: func(url string) bool {
			return qt.QDesktopServices_OpenUrl(qt.NewQUrl3(url))
		},
	}

	l.Insert("play", qt.NewQVariant14(""))
	l.Insert("message", qt.NewQVariant14(""))
	l.Insert("messageId", qt.NewQVariant4(0))

	// Only fires for changes made from QML, not for our own Inserts.
	l.OnValueChanged(func(key string, value *qt.QVariant) {
		if key != "play" || value.ToString() == "" {
			return
		}

		mode := value.ToString()

		// Cleared, so playing the same way again is still a change.
		l.Insert(key, qt.NewQVariant14(""))
		l.Play(mode == "mods")
	})

	return l
}

// Play starts Deadlock, with or without mods.
func (l *gameLauncher) Play(withMods bool) {
	// TODO: put the mods that are turned on into the game first, once
	// installing mods into the game exists. Until then, both start it as it is.
	args := game.SplitOptions(l.launchOptions())

	slog.Info("starting Deadlock", "mods", withMods, "args", args)

	err := l.launch(args)
	if errors.Is(err, game.ErrNoSteam) {
		// Whatever opens steam:// links may still find it, e.g. a Steam
		// installed somewhere unusual.
		url := game.SteamURL(args)

		if l.openURL(url) {
			err = nil
		} else {
			slog.Error("failed to open a steam:// link", "url", url)
		}
	}

	if err != nil {
		slog.Error("failed to start Deadlock", "err", err)
		l.notify(tr("Couldn’t start Deadlock. Check that Steam is installed."))

		return
	}

	l.notify(tr("Starting Deadlock"))
}

func (l *gameLauncher) notify(message string) {
	l.messages++
	l.Insert("message", qt.NewQVariant14(message))
	l.Insert("messageId", qt.NewQVariant4(l.messages))
}
