package main

import (
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
	"github.com/treejadey/lockdeader/game"
)

// settingsPage backs the settings page. QML changes a setting by writing its
// key:
//
//	showNsfw: whether mods GameBanana marks as NSFW are listed
//	launchOptions: what's added to Deadlock's command line
//	chooseGameFolder: a folder the user picked as Deadlock's, as a URL.
//	                  It's only taken if it is Deadlock's (see game.Check);
//	                  if not, "gameFolderError" says so.
//	findGame:         true: looks for Deadlock in Steam's libraries, if
//	                  there's no folder for it yet
//
// Go fills in "gameFolder", Deadlock's folder ("" for none), "dataDir", the
// path of the data directory, and URLs of both ("gameFolderUrl",
// "dataDirUrl"), for opening them in the file manager.
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
	p.Insert("chooseGameFolder", qt.NewQVariant14(""))
	p.Insert("findGame", qt.NewQVariant8(false))
	p.Insert("gameFolderError", qt.NewQVariant14(""))
	p.showGameFolder()
	p.Insert("dataDir", qt.NewQVariant14(dataDir))
	p.Insert("dataDirUrl", qt.NewQVariant14(qt.QUrl_FromLocalFile(dataDir).ToString()))

	// Only fires for changes made from QML, not for our own Inserts.
	p.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "showNsfw":
			p.SetShowNSFW(value.ToBool())
		case "launchOptions":
			p.SetLaunchOptions(value.ToString())
		case "chooseGameFolder":
			url := value.ToString()
			if url == "" {
				return
			}

			// Cleared, so picking the same folder again is still a change.
			p.Insert(key, qt.NewQVariant14(""))
			p.ChooseGameFolder(qt.NewQUrl3(url).ToLocalFile())
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
