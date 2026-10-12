package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/filedialog"
	"github.com/treejadey/lockdeader/game"
	"github.com/treejadey/lockdeader/install"
	"github.com/treejadey/lockdeader/notification"
)

func main() {
	guiApp := qt.NewQGuiApplication(os.Args)

	if err := loadFonts(); err != nil {
		slog.Error("failed to load the bundled fonts", "err", err)
		os.Exit(1)
	}

	if err := setWindowIcon(); err != nil {
		slog.Warn("failed to set the window icon", "err", err)
	}

	installTranslations()

	dirs, err := appdata.Setup()
	if err != nil {
		slog.Error("failed to set up the data directory", "err", err)
		os.Exit(1)
	}

	slog.Info("using data directory", "path", dirs.Root)

	// Set once the window is up.
	var showWindow func()

	if !claimInstance(dirs.Root, func() {
		if showWindow != nil {
			showWindow()
		}
	}) {
		slog.Info("the app is already running, so it was asked to show its window instead")
		os.Exit(0)
	}

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

	uiState, err := db.UIState(ctx)
	if err != nil {
		slog.Error("failed to read where the app was left", "err", err)
	}

	var ui *uiStateBridge

	saveUISoon, flushUI := debounced(func() {
		if err := db.SetUIState(ctx, ui.State()); err != nil {
			slog.Error("failed to save where the app was left", "err", err)
		}
	})

	ui = newUIStateBridge(uiState, saveUISoon)

	app := qml.NewQQmlPropertyMap()
	// For numbers that shouldn't shift as they change.
	app.Insert("monospaceFont", qt.NewQVariant14(monospaceFontFamily))
	// Goes up by one each time another copy of the app asks to show the window.
	app.Insert("showRequests", qt.NewQVariant4(0))

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

	// Puts the mods that are on into the game after each change to them.
	deployer := newDeployer(db, store, func() string { return settings.Game.Path },
		func() bool { return settings.Game.LoadOtherMods }, dirs.GameInfoBackup())
	installed.onChange = deployer.ApplySoon

	// Mods other tools put in the game, shown until they're taken over.
	otherMods := newOtherMods(client, store, installed, func() string { return settings.Game.Path })
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

	categories.onSelected = ui.SetCategory
	details.onSelected = ui.SetOpenMod

	// A game folder that's stopped being Deadlock's, e.g. once it's
	// uninstalled, is dropped, and Deadlock is looked for again.
	if path := settings.Game.Path; path != "" {
		folder, err := game.Check(path)
		if err != nil {
			slog.Warn("the game folder isn't Deadlock's any more", "path", path)
		}

		if folder != path {
			settings.Game.Path = folder
			settingsChanged()
		}
	}

	settingsPage := newSettingsPage(settings.Browse, settings.Game, dirs.Root, func(show bool) {
		settings.Browse.ShowNSFW = show
		settingsChanged()

		if mods.SetHideNSFW(!show) {
			listingChanged()
		}
	}, func(gameSettings appdata.GameSettings) {
		changedFolder := gameSettings.Path != settings.Game.Path
		settings.Game = gameSettings
		settingsChanged()

		if changedFolder {
			deployer.ApplySoon()
			otherMods.Scan()
		}
	})

	settingsPage.FindGame()

	launcher := newGameLauncher(func() string { return settings.Game.LaunchOptions },
		func() string { return settings.Game.Path }, deployer.Prepare)

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
	engine.RootContext().SetContextProperty("otherMods", otherMods.QObject)
	engine.RootContext().SetContextProperty("otherModsModel", otherMods.Model.QObject)
	engine.RootContext().SetContextProperty("categories", categories.QObject)
	engine.RootContext().SetContextProperty("search", search.QObject)
	engine.RootContext().SetContextProperty("clipboard", clipboard.QObject)
	engine.RootContext().SetContextProperty("soundPreview", soundPreview.QObject)
	engine.RootContext().SetContextProperty("settings", settingsPage.QObject)
	engine.RootContext().SetContextProperty("uiState", ui.QObject)
	engine.RootContext().SetContextProperty("launcher", launcher.QObject)
	engine.RootContext().SetContextProperty("retryAction", retryAction.QObject)
	engine.RootContext().SetContextProperty("retryCategoriesAction", retryCategoriesAction.QObject)

	engine.Load(qt.QUrl_FromLocalFile("qml/Main.qml"))

	if len(engine.RootObjects()) == 0 {
		slog.Error("failed to load qml/Main.qml")
		os.Exit(1)
	}

	// The root object is the window: QObject is QWindow's first base, so the
	// pointer is the same. Folder dialogs open over it where the platform
	// allows.
	mainWindow := qt.UnsafeNewQWindow(engine.RootObjects()[0].UnsafePointer())

	settingsPage.window = func() filedialog.Window {
		return filedialog.Window{ID: mainWindow.WinId(), X11: qt.QGuiApplication_PlatformName() == "xcb"}
	}

	// Closing the window while mods are installing only hides it, so they can
	// finish; then the app quits. A notification tells of any that fail in
	// the meantime.
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	windowOpen := true

	quitIfDone := func() {
		if !windowOpen && installed.Idle() && deployer.Idle() {
			qt.QCoreApplication_Quit()
		}
	}

	guiApp.OnLastWindowClosed(func() {
		windowOpen = false

		// Nothing plays from a closed window.
		soundPreview.Load("")

		// A change still waiting to reach the game does so before quitting.
		deployer.Flush()
		quitIfDone()
	})

	installed.onIdle = quitIfDone

	// Why the mods couldn't be put into the game shows on the installed
	// page, until they can. A running game isn't a problem: the change
	// waits for the next apply.
	deployer.onDone = func(err error) {
		if !errors.Is(err, deploy.ErrGameRunning) {
			installed.SetApplyError(err)
		}

		quitIfDone()
	}

	installed.onFailed = func(name string) {
		if windowOpen {
			return
		}

		title := fmt.Sprintf(tr("Couldn’t install %s"), name)
		body := tr("Try installing it again in Lockdeader")

		go func() {
			if err := notification.Send(title, body); err != nil {
				slog.Warn("failed to show a notification", "err", err)
			}
		}()
	}

	showRequests := 0

	showWindow = func() {
		windowOpen = true

		showRequests++
		app.Insert("showRequests", qt.NewQVariant4(showRequests))
	}

	// Reopen where the user left off: in the same category, and scrolled to
	// the same mod with the same one open, if they're near enough the top to
	// find. If not, the list starts at the top, with nothing open. The grid
	// requests more pages as it scrolls.
	saved := ui.State()
	categories.Restore(banana.Section(saved.Section), saved.CategoryID)

	var lookFor []int

	for _, id := range []int{saved.ModsTopMod, saved.OpenMod} {
		if id != 0 {
			lookFor = append(lookFor, id)
		}
	}

	mods.Seek(lookFor, restorePages, func(rows map[int]int) {
		if _, ok := rows[saved.OpenMod]; ok && saved.OpenMod != 0 {
			details.Select(saved.OpenMod)
		} else {
			ui.SetOpenMod(0)
		}

		if row, ok := rows[saved.ModsTopMod]; ok && saved.ModsTopMod != 0 {
			ui.RestoreModsScroll(row, saved.ModsTopOffset)
		} else {
			ui.RestoreModsScroll(-1, 0)
		}
	})

	categories.Load()

	// Lockdeader's lines in gameinfo.gi are only for its own launches, so
	// they come out on startup; and the mods go into the game again, in
	// case a previous session left that unfinished.
	if settings.Game.Path != "" {
		deployer.ResetSoon()
		otherMods.Scan()
	}

	// While Deadlock runs, the Play buttons are disabled. Once it closes, the
	// lines come out of gameinfo.gi, so starting it from Steam runs it
	// without mods, and changes made meanwhile go in.
	watcher := newGameWatcher(func(running bool) {
		launcher.SetGameRunning(running)

		if !running {
			deployer.ResetSoon()
		}
	})
	watcher.Start()

	code := qt.QGuiApplication_Exec()

	// Undo is too late once the app has gone.
	installed.FinishUninstall()

	// Save anything still pending, and back up the settings if they changed.
	flushSettings()
	flushAudio()
	flushUI()

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
