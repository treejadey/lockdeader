package main

import (
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
)

// The user's answers to the restore dialog, besides a backup's slot.
const (
	restoreQuit   = -1 // quit without touching anything
	restoreIgnore = 0  // carry on without the settings file
)

// restoreDialog backs qml/RestoreDialog.qml, which asks whether to restore a
// backup of the settings file when it fails to load, like Chatterino does.
// Go fills in:
//
//	error:          why the settings file failed to load
//	backups:        {slot, modified (Unix seconds), url} of each backup that
//	                can be restored, newest first
//	damaged:        whether there are other backups, which can't be restored
//	settingsDirUrl: the folder they're all in, as a file URL
//
// QML writes the user's answer to "choice": a backup's slot to restore it,
// restoreIgnore or restoreQuit.
// All methods must be called on the Qt main thread.
type restoreDialog struct {
	*qml.QQmlPropertyMap

	// Called with the user's answer.
	onChoice func(choice int)
}

func newRestoreDialog(loadErr error, backups []appdata.SettingsBackup, settingsDir string, onChoice func(choice int)) *restoreDialog {
	d := &restoreDialog{QQmlPropertyMap: qml.NewQQmlPropertyMap(), onChoice: onChoice}

	list := make([]qt.QVariant, 0, len(backups))
	damaged := false
	for _, b := range backups {
		if b.Err != nil {
			damaged = true
			continue
		}
		list = append(list, *qt.NewQVariant20(map[string]qt.QVariant{
			"slot":     *qt.NewQVariant4(b.Slot),
			"modified": *qt.NewQVariant6(b.Modified.Unix()),
			"url":      *qt.NewQVariant14(qt.QUrl_FromLocalFile(b.Path).ToString()),
		}))
	}
	d.Insert("error", qt.NewQVariant14(loadErr.Error()))
	d.Insert("backups", qt.NewQVariant43(list))
	d.Insert("damaged", qt.NewQVariant8(damaged))
	d.Insert("settingsDirUrl", qt.NewQVariant14(qt.QUrl_FromLocalFile(settingsDir).ToString()))
	d.Insert("choice", qt.NewQVariant4(restoreQuit))

	// Only fires for changes made from QML, not for our own Inserts.
	d.OnValueChanged(func(key string, value *qt.QVariant) {
		if key == "choice" && d.onChoice != nil {
			d.onChoice(value.ToInt())
		}
	})
	return d
}

// askToRestoreSettings shows the restore dialog and waits for the user's
// answer: the slot of the backup to restore, restoreIgnore or restoreQuit. It
// returns restoreIgnore if there are no backups that can be restored, or the
// dialog can't be shown.
//
// It runs before the app's window is shown, so it runs an event loop of its
// own.
func askToRestoreSettings(loadErr error, backups []appdata.SettingsBackup, settingsDir string) int {
	restorable := false
	for _, b := range backups {
		restorable = restorable || b.Err == nil
	}
	if !restorable {
		return restoreIgnore
	}

	loop := qt.NewQEventLoop()
	defer loop.Delete()
	// Closing the window answers restoreQuit; don't let it quit the app first.
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	defer qt.QGuiApplication_SetQuitOnLastWindowClosed(true)

	choice := restoreQuit
	dialog := newRestoreDialog(loadErr, backups, settingsDir, func(c int) {
		choice = c
		loop.Quit()
	})
	defer dialog.Delete()

	// The main window's app has more in it, but this is all the dialog uses.
	app := qml.NewQQmlPropertyMap()
	defer app.Delete()
	app.Insert("monospaceFont", qt.NewQVariant14(monospaceFontFamily))

	engine := qml.NewQQmlApplicationEngine()
	defer engine.Delete()
	engine.RootContext().SetContextProperty("app", app.QObject)
	engine.RootContext().SetContextProperty("restore", dialog.QObject)
	engine.Load(qt.QUrl_FromLocalFile("qml/RestoreDialog.qml"))
	if len(engine.RootObjects()) == 0 {
		slog.Error("failed to load qml/RestoreDialog.qml")
		return restoreIgnore
	}

	loop.Exec()
	return choice
}
