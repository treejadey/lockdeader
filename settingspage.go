package main

import (
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
	"github.com/treejadey/lockdeader/filedialog"
	"github.com/treejadey/lockdeader/game"
)

// settingsPage backs the settings page. QML changes a setting by writing its
// key:
//
//	showNsfw: whether mods GameBanana marks as NSFW are listed
//	launchOptions: what's added to Deadlock's command line
//	loadOtherMods: whether mods installed outside Lockdeader load too
//	chooseGameFolder: true: asks the user for Deadlock's folder with the
//	                  system's dialog. The folder is only taken if it is
//	                  Deadlock's (see game.Check); if not, "gameFolderError"
//	                  says so. "choosingGameFolder" is true while it's open.
//	findGame:         true: looks for Deadlock in Steam's libraries, if
//	                  there's no folder for it yet
//
// Go fills in "gameFolder", Deadlock's folder ("" for none), "dataDir", the
// path of the data directory, and URLs of both ("gameFolderUrl",
// "dataDirUrl"), for opening them in the file manager, and "buildInfo", the
// version and what it runs on (see buildInfo), as rich text.
//
// All methods must be called on the Qt main thread.
type settingsPage struct {
	*qml.QQmlPropertyMap

	browse appdata.BrowseSettings
	game   appdata.GameSettings

	onShowNSFWChange func(show bool)                     // may be nil
	onGameChange     func(settings appdata.GameSettings) // may be nil

	// Looks for Deadlock; game.Find, other than in tests.
	find    func() string
	finding bool

	// The window the folder dialog opens over. May be nil.
	window   func() filedialog.Window
	choosing bool
}

func newSettingsPage(browse appdata.BrowseSettings, gameSettings appdata.GameSettings, dataDir string,
	onShowNSFWChange func(show bool), onGameChange func(settings appdata.GameSettings)) *settingsPage {
	p := &settingsPage{
		QQmlPropertyMap:  qml.NewQQmlPropertyMap(),
		browse:           browse,
		game:             gameSettings,
		onShowNSFWChange: onShowNSFWChange,
		onGameChange:     onGameChange,
		find:             game.Find,
	}

	p.Insert("showNsfw", qt.NewQVariant8(browse.ShowNSFW))
	p.Insert("launchOptions", qt.NewQVariant14(gameSettings.LaunchOptions))
	p.Insert("loadOtherMods", qt.NewQVariant8(gameSettings.LoadOtherMods))
	p.Insert("chooseGameFolder", qt.NewQVariant8(false))
	p.Insert("choosingGameFolder", qt.NewQVariant8(false))
	p.Insert("findGame", qt.NewQVariant8(false))
	p.Insert("gameFolderError", qt.NewQVariant14(""))
	p.showGameFolder()
	p.Insert("dataDir", qt.NewQVariant14(dataDir))
	p.Insert("dataDirUrl", qt.NewQVariant14(qt.QUrl_FromLocalFile(dataDir).ToString()))
	p.Insert("buildInfo", qt.NewQVariant14(buildInfo()))

	// Only fires for changes made from QML, not for our own Inserts.
	p.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "showNsfw":
			p.SetShowNSFW(value.ToBool())
		case "launchOptions":
			p.SetLaunchOptions(value.ToString())
		case "loadOtherMods":
			p.SetLoadOtherMods(value.ToBool())
		case "chooseGameFolder":
			if value.ToBool() {
				p.Insert(key, qt.NewQVariant8(false))
				p.AskForGameFolder()
			}
		case "findGame":
			if value.ToBool() {
				p.Insert(key, qt.NewQVariant8(false))
				p.FindGame()
			}
		}
	})

	return p
}

func (p *settingsPage) SetShowNSFW(show bool) {
	p.Insert("showNsfw", qt.NewQVariant8(show))

	if show == p.browse.ShowNSFW {
		return
	}

	p.browse.ShowNSFW = show

	if p.onShowNSFWChange != nil {
		p.onShowNSFWChange(show)
	}
}

func (p *settingsPage) SetLaunchOptions(options string) {
	p.Insert("launchOptions", qt.NewQVariant14(options))

	if options == p.game.LaunchOptions {
		return
	}

	p.game.LaunchOptions = options
	p.gameChanged()
}

func (p *settingsPage) SetLoadOtherMods(load bool) {
	p.Insert("loadOtherMods", qt.NewQVariant8(load))

	if load == p.game.LoadOtherMods {
		return
	}

	p.game.LoadOtherMods = load
	p.gameChanged()
}

// AskForGameFolder opens the system's folder dialog, starting in Deadlock's
// folder if there is one, and takes the folder picked as ChooseGameFolder
// does.
func (p *settingsPage) AskForGameFolder() {
	if p.choosing {
		return
	}

	var window filedialog.Window

	if p.window != nil {
		window = p.window()
	}

	title, start := tr("Choose Deadlock’s Folder"), p.game.Path
	p.setChoosing(true)

	go func() {
		folder, err := filedialog.ChooseFolder(window, title, start)

		mainthread.Start(func() {
			p.setChoosing(false)

			if err != nil {
				slog.Error("failed to open the folder dialog", "err", err)
				p.Insert("gameFolderError", qt.NewQVariant14(tr("The folder dialog couldn’t be opened.")))

				return
			}

			// Cancelled.
			if folder == "" {
				return
			}

			p.ChooseGameFolder(folder)
		})
	}()
}

func (p *settingsPage) setChoosing(choosing bool) {
	p.choosing = choosing
	p.Insert("choosingGameFolder", qt.NewQVariant8(choosing))
}

// ChooseGameFolder takes dir as Deadlock's folder if it is, or is inside it.
// If it isn't, the folder Deadlock had stays, and "gameFolderError" says why.
func (p *settingsPage) ChooseGameFolder(dir string) {
	folder, err := game.Check(dir)
	if err != nil {
		slog.Info("the folder picked isn't Deadlock's", "path", dir)
		p.Insert("gameFolderError", qt.NewQVariant14(
			tr("That folder isn’t Deadlock’s. Choose the one Steam installed it in, usually steamapps/common/Deadlock.")))

		return
	}

	p.setGameFolder(folder)
}

// FindGame looks for Deadlock in the background, if there's no folder for it
// yet. Not finding it leaves the folder empty, without an error: the user
// hasn't asked for anything.
func (p *settingsPage) FindGame() {
	if p.game.Path != "" || p.finding {
		return
	}

	p.finding = true

	go func() {
		folder := p.find()

		mainthread.Start(func() {
			p.finding = false

			// Unless the user picked one in the meantime.
			if folder == "" || p.game.Path != "" {
				return
			}

			slog.Info("found Deadlock", "path", folder)
			p.setGameFolder(folder)
		})
	}()
}

func (p *settingsPage) setGameFolder(folder string) {
	p.Insert("gameFolderError", qt.NewQVariant14(""))

	if folder == p.game.Path {
		return
	}

	p.game.Path = folder
	p.showGameFolder()
	p.gameChanged()
}

func (p *settingsPage) showGameFolder() {
	url := ""

	if p.game.Path != "" {
		url = qt.QUrl_FromLocalFile(p.game.Path).ToString()
	}

	p.Insert("gameFolder", qt.NewQVariant14(p.game.Path))
	p.Insert("gameFolderUrl", qt.NewQVariant14(url))
}

func (p *settingsPage) gameChanged() {
	if p.onGameChange != nil {
		p.onGameChange(p.game)
	}
}
