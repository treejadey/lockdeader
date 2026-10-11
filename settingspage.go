package main

import (
	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
)

// settingsPage backs the settings page. QML changes a setting by writing its
// key:
//
//	showNsfw: whether mods GameBanana marks as NSFW are listed
//
// Go fills in "dataDir", the path of the data directory, and "dataDirUrl",
// the same as a file URL, for opening it in the file manager.
//
// All methods must be called on the Qt main thread.
type settingsPage struct {
	*qml.QQmlPropertyMap

	browse appdata.BrowseSettings

	onShowNSFWChange func(show bool) // may be nil
}

func newSettingsPage(browse appdata.BrowseSettings, dataDir string, onShowNSFWChange func(show bool)) *settingsPage {
	p := &settingsPage{
		QQmlPropertyMap:  qml.NewQQmlPropertyMap(),
		browse:           browse,
		onShowNSFWChange: onShowNSFWChange,
	}

	p.Insert("showNsfw", qt.NewQVariant8(browse.ShowNSFW))
	p.Insert("dataDir", qt.NewQVariant14(dataDir))
	p.Insert("dataDirUrl", qt.NewQVariant14(qt.QUrl_FromLocalFile(dataDir).ToString()))

	// Only fires for changes made from QML, not for our own Inserts.
	p.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "showNsfw":
			p.SetShowNSFW(value.ToBool())
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
