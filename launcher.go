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
// Go first gets the game ready (see deployer.Prepare), then adds the launch
// options from the settings. As in the clipboard bridge, "message" says how
// it went, and "messageId" goes up by one with each. "messageAction" is
// "settings" for a message that should offer to open the settings, or "".
//
// All methods must be called on the Qt main thread.
type gameLauncher struct {
	*qml.QQmlPropertyMap

	launchOptions func() string
	gameFolder    func() string // "" while there's none
	// Puts the mods into the game, or takes them out; deployer.Prepare.
	prepare func(withMods bool, then func(error))

	// game.Launch and opening a URL, other than in tests.
	launch  func(args []string) error
	openURL func(url string) bool

	preparing bool
	messages  int
}

func newGameLauncher(launchOptions, gameFolder func() string, prepare func(withMods bool, then func(error))) *gameLauncher {
	l := &gameLauncher{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		launchOptions:   launchOptions,
		gameFolder:      gameFolder,
		prepare:         prepare,
		launch:          game.Launch,
		openURL: func(url string) bool {
			return qt.QDesktopServices_OpenUrl(qt.NewQUrl3(url))
		},
	}

	l.Insert("play", qt.NewQVariant14(""))
	l.Insert("message", qt.NewQVariant14(""))
	l.Insert("messageAction", qt.NewQVariant14(""))
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

// Play starts Deadlock, with or without mods, once the game is ready. If it
// can't be made ready, it isn't started: the message says why.
func (l *gameLauncher) Play(withMods bool) {
	if l.preparing {
		return
	}

	if l.gameFolder() == "" {
		l.notify(tr("Choose Deadlock’s folder first."), "settings")

		return
	}

	l.preparing = true

	l.prepare(withMods, func(err error) {
		l.preparing = false

		// The reason alone: a toast is short, and the installed page says
		// more about mods not reaching the game.
		if err != nil {
			l.notify(errorMessage(err), "")

			return
		}

		l.start(withMods)
	})
}

func (l *gameLauncher) start(withMods bool) {
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
		l.notify(tr("Couldn’t start Deadlock. Check that Steam is installed."), "")

		return
	}

	if withMods {
		l.notify(tr("Starting Deadlock with mods"), "")
	} else {
		l.notify(tr("Starting Deadlock without mods"), "")
	}
}

func (l *gameLauncher) notify(message, action string) {
	l.messages++
	l.Insert("message", qt.NewQVariant14(message))
	l.Insert("messageAction", qt.NewQVariant14(action))
	l.Insert("messageId", qt.NewQVariant4(l.messages))
}
