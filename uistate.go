package main

import (
	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
)

// The mods page looks this many pages into its list for where the user left
// off. Further down, it starts at the top instead: each page is a request to
// GameBanana.
const restorePages = 3

// uiStateBridge remembers where the user left the app (see appdb.UIState), so
// it reopens there. QML writes these keys as they change:
//
//	page, modsView, installedView
//	categoryPath:  the categories opened in the sidebar, as "Section/id",
//	               comma-separated
//	modsTopMod, modsTopOffset: the first mod on screen and how far past its top
//	installedTop, installedTopOffset: the same, by row, on the installed page
//	settingsY
//
// Go keeps the category and the open mod up to date itself. Once it knows
// where the mods page should scroll to, it sets "modsRestoreRow" (-1 for the
// top) and "modsRestoreOffset", then bumps "modsRestoreId".
//
// All methods must be called on the Qt main thread.
type uiStateBridge struct {
	*qml.QQmlPropertyMap

	state    appdb.UIState
	onChange func() // may be nil

	restores int
}

func newUIStateBridge(state appdb.UIState, onChange func()) *uiStateBridge {
	b := &uiStateBridge{QQmlPropertyMap: qml.NewQQmlPropertyMap(), state: state, onChange: onChange}

	ints := map[string]int{
		"page":               state.Page,
		"modsView":           state.ModsView,
		"installedView":      state.InstalledView,
		"modsTopMod":         state.ModsTopMod,
		"modsTopOffset":      state.ModsTopOffset,
		"installedTop":       state.InstalledTop,
		"installedTopOffset": state.InstalledTopOffset,
		"settingsY":          state.SettingsY,
		"modsRestoreRow":     -1,
		"modsRestoreOffset":  0,
		"modsRestoreId":      0,
	}

	for key, value := range ints {
		b.Insert(key, qt.NewQVariant4(value))
	}

	b.Insert("categoryPath", qt.NewQVariant14(state.CategoryPath))

	// Only fires for changes made from QML, not for our own Inserts.
	b.OnValueChanged(func(key string, value *qt.QVariant) {
		fields := map[string]*int{
			"page":               &b.state.Page,
			"modsView":           &b.state.ModsView,
			"installedView":      &b.state.InstalledView,
			"modsTopMod":         &b.state.ModsTopMod,
			"modsTopOffset":      &b.state.ModsTopOffset,
			"installedTop":       &b.state.InstalledTop,
			"installedTopOffset": &b.state.InstalledTopOffset,
			"settingsY":          &b.state.SettingsY,
		}

		if field, ok := fields[key]; ok {
			*field = max(0, value.ToInt())
		} else if key == "categoryPath" {
			b.state.CategoryPath = value.ToString()
		} else {
			return
		}

		b.changed()
	})

	return b
}

func (b *uiStateBridge) State() appdb.UIState {
	return b.state
}

func (b *uiStateBridge) SetCategory(section banana.Section, id int) {
	b.state.Section = string(section)
	b.state.CategoryID = id
	b.changed()
}

func (b *uiStateBridge) SetOpenMod(id int) {
	b.state.OpenMod = id
	b.changed()
}

// RestoreModsScroll tells the mods page where to scroll to; a row of -1
// leaves it at the top, and forgets the mod that was there, so it isn't
// looked for again next time.
func (b *uiStateBridge) RestoreModsScroll(row, offset int) {
	if row < 0 {
		b.state.ModsTopMod, b.state.ModsTopOffset = 0, 0
		b.Insert("modsTopMod", qt.NewQVariant4(0))
		b.Insert("modsTopOffset", qt.NewQVariant4(0))
		b.changed()
	}

	b.restores++
	b.Insert("modsRestoreRow", qt.NewQVariant4(row))
	b.Insert("modsRestoreOffset", qt.NewQVariant4(offset))
	b.Insert("modsRestoreId", qt.NewQVariant4(b.restores))
}

func (b *uiStateBridge) changed() {
	if b.onChange != nil {
		b.onChange()
	}
}
