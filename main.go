package main

import (
	"context"
	"log/slog"
	"os"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/install"
)

func main() {
	qt.NewQGuiApplication(os.Args)

	if err := loadFonts(); err != nil {
		slog.Error("failed to load the bundled fonts", "err", err)
		os.Exit(1)
	}

	dirs, err := appdata.Setup()
	if err != nil {
		slog.Error("failed to set up the data directory", "err", err)
		os.Exit(1)
	}

	slog.Info("using data directory", "path", dirs.Root)

	// Before any QML is loaded, including the restore dialog's.
	useNativeTextRendering()

	settings, err := appdata.LoadSettings(dirs.SettingsFile())

	// Offer to restore a backup until the file loads, or the user would
	// rather carry on without it.
	for err != nil {
		slog.Warn("failed to load the settings", "err", err)

		choice := askToRestoreSettings(err, appdata.FindSettingsBackups(dirs.SettingsFile()), dirs.Settings)
		if choice == restoreQuit {
			os.Exit(0)
		}

		if choice == restoreIgnore {
			break
		}

		if err = appdata.RestoreSettingsBackup(dirs.SettingsFile(), choice); err != nil {
			continue
		}

		slog.Info("restored a backup of the settings", "slot", choice)
		settings, err = appdata.LoadSettings(dirs.SettingsFile())
	}

	// Settings changed in the app are saved, but never over a file that
	// couldn't be read: the user may want to fix it instead.
	canSave := err == nil
	if err != nil {
		slog.Warn("using default settings", "err", err)
	}

	// What the settings file held before this session, which is backed up
	// on quit if the session changed it.
	var startupSettings []byte

	if canSave {
		if startupSettings, err = os.ReadFile(dirs.SettingsFile()); err != nil {
			slog.Warn("settings won't be backed up", "err", err)
		}
	}

	saveSettingsSoon, flushSettings := debounced(func() {
		if err := appdata.SaveSettings(dirs.SettingsFile(), settings); err != nil {
			slog.Error("failed to save the settings", "err", err)
		}
	})

	settingsChanged := func() {
		if canSave {
			saveSettingsSoon()
		}
	}

	// Like the settings, a database that can't be opened is never written
	// over, e.g. one a newer version has upgraded: this session carries on
	// with one in memory, and forgets it all on quit.
	ctx := context.Background()
	db, err := appdb.Open(dirs.DatabaseFile())
	dbIsSaved := err == nil
	if err != nil {
		slog.Error("failed to open the database; nothing will be remembered this session", "err", err)

		if db, err = appdb.OpenMemory(); err != nil {
			slog.Error("failed to create a database in memory", "err", err)
			os.Exit(1)
		}
	}

	// Older versions kept the volume in the settings file. Move it here,
	// then save the settings without it.
	if legacy, ok, _ := appdata.ReadLegacyAudio(dirs.SettingsFile()); ok && canSave && dbIsSaved {
		if _, err := db.ImportAudioState(ctx, appdb.AudioState{Volume: legacy.Volume, Muted: legacy.Muted}); err != nil {
			slog.Error("failed to move the volume to the database", "err", err)
		} else {
			saveSettingsSoon()
		}
	}

	audioState, err := db.AudioState(ctx)
	if err != nil {
		slog.Error("failed to read the volume", "err", err)
	}

	saveAudioSoon, flushAudio := debounced(func() {
		if err := db.SetAudioState(ctx, audioState); err != nil {
			slog.Error("failed to save the volume", "err", err)
		}
	})

	app := qml.NewQQmlPropertyMap()
	// For numbers that shouldn't shift as they change.
	app.Insert("monospaceFont", qt.NewQVariant14(monospaceFontFamily))

	client := banana.NewClient()
	mods := newModListModel(client, app)
	mods.SetHideNSFW(!settings.Browse.ShowNSFW)

	// Installs the app quit during leave files behind; nothing is installing yet.
	store := install.Store{Dir: dirs.ModCache}
	if err := store.CleanUp(); err != nil {
		slog.Warn("failed to clean up after unfinished installs", "err", err)
	}

	// Only the real database says which VPKs are still needed.
	if dbIsSaved {
		if err := removeUnusedVPKs(ctx, db, store); err != nil {
			slog.Warn("failed to delete VPKs that are no longer needed", "err", err)
		}
	}

	installed := newInstalledMods(client, db, store)
	details := newModDetails(client, mods.Section, mods.Lookup, installed.Install)
	clipboard := newClipboardBridge(dirs.ImageCache, client)

	soundPreview := newSoundPreview(client, audioState, func(volume float64, muted bool) {
		audioState = appdb.AudioState{Volume: volume, Muted: muted}
		saveAudioSoon()
	})

	// Switching category or searching closes the open mod, back to the
	// "select a mod" hint, and stops the sound being played: its row is gone.
	listingChanged := func() {
		details.Select(0)
		soundPreview.Load("")
	}

	categories := newCategoryBrowser(client, mods, listingChanged)
	search := newModSearch(mods, details.Select, listingChanged)

	settingsPage := newSettingsPage(settings.Browse, dirs.Root, func(show bool) {
		settings.Browse.ShowNSFW = show
		settingsChanged()

		if mods.SetHideNSFW(!show) {
			listingChanged()
		}
	})

	retryAction := qt.NewQAction()
	retryAction.OnTriggered(mods.Retry)
	retryCategoriesAction := qt.NewQAction()
	retryCategoriesAction.OnTriggered(categories.Load)

	engine := qml.NewQQmlApplicationEngine()

	installImageCache(engine.QQmlEngine, dirs.ImageCache,
		int64(settings.Cache.ImagesMaxSizeMB)<<20, []string{banana.ImageHost})

	engine.RootContext().SetContextProperty("app", app.QObject)
	engine.RootContext().SetContextProperty("mods", mods.QObject)
	engine.RootContext().SetContextProperty("details", details.QObject)
	engine.RootContext().SetContextProperty("installed", installed.QObject)
	engine.RootContext().SetContextProperty("installedMods", installed.Filtered.QObject)
	engine.RootContext().SetContextProperty("categories", categories.QObject)
	engine.RootContext().SetContextProperty("search", search.QObject)
	engine.RootContext().SetContextProperty("clipboard", clipboard.QObject)
	engine.RootContext().SetContextProperty("soundPreview", soundPreview.QObject)
	engine.RootContext().SetContextProperty("settings", settingsPage.QObject)
	engine.RootContext().SetContextProperty("retryAction", retryAction.QObject)
	engine.RootContext().SetContextProperty("retryCategoriesAction", retryCategoriesAction.QObject)

	engine.Load(qt.QUrl_FromLocalFile("qml/Main.qml"))

	if len(engine.RootObjects()) == 0 {
		slog.Error("failed to load qml/Main.qml")
		os.Exit(1)
	}

	// Load the first page; the grid requests the rest as it scrolls.
	mods.FetchNextPage()
	categories.Load()

	code := qt.QGuiApplication_Exec()

	// Undo is too late once the app has gone.
	installed.FinishUninstall()

	// Save anything still pending, and back up the settings if they changed.
	flushSettings()
	flushAudio()

	if err := db.Close(); err != nil {
		slog.Error("failed to close the database", "err", err)
	}

	if startupSettings != nil {
		if err := appdata.BackUpSettings(dirs.SettingsFile(), startupSettings); err != nil {
			slog.Error("failed to back up the settings", "err", err)
		}
	}

	os.Exit(code)
}

func debounced(f func()) (save, flush func()) {
	timer := qt.NewQTimer()
	timer.SetSingleShot(true)
	timer.SetInterval(500)
	timer.OnTimeout(f)

	save = func() { timer.Start2() }

	flush = func() {
		if timer.IsActive() {
			timer.Stop()

			f()
		}
	}

	return save, flush
}
