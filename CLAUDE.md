# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Lockdeader is a desktop mod manager for the game Deadlock (a work in progress: it browses mods on GameBanana, downloads them into its own store, and puts the ones that are turned on into the game). It is written in Go with a Qt 6 Quick/QML UI, using the [miqt](https://github.com/mappu/miqt) bindings.

Windows matters as much as Linux.
- Anything platform-specific needs a Windows side as well as a Linux one, e.g. files with `_linux.go` and `_windows.go` suffixes, as in `notification/`. A stub that does nothing is only for other platforms.
- Prefer Qt or pure Go where they work on both, e.g. `QLocalServer`, which is a socket on Linux and a named pipe on Windows.
- Check that new non-cgo packages build for Windows: `GOOS=windows go vet ./<package>`.

## Commands

```sh
go build -o lockdeader .     # needs Qt 6 dev packages (pkg-config: Qt6Qml, Qt6Network, Qt6Quick) and a C++ compiler; uses cgo
./lockdeader                 # runs from anywhere: qml/ is embedded, so QML changes need a rebuild
go test ./...                # all tests
go test ./banana -run TestListModsDecode   # a single test
go vet ./... && gofmt -l .
go run ./cmd/translations extract   # after changing UI text: updates po/lockdeader.pot and po/*.po
go run ./cmd/translations check     # out-of-date template, mistakes in translations
git config core.hooksPath .githooks # once per clone: the pre-commit hook runs that check on what's staged
```

- To run headless (e.g. to smoke-test that QML loads), set `QT_QPA_PLATFORM=offscreen`; add `QT_FORCE_STDERR_LOGGING=1` to see QML warnings.
  - The offscreen platform uses Qt's software renderer, which doesn't draw `MultiEffect` layers, so mod cards, rounded images and shadows come out blank.
  - For screenshots that match the real app, run it inside a nested, invisible KWin session instead:

    ```sh
    kwin_wayland --virtual --no-lockscreen --width 1625 --height 1000 --socket ld-test --exit-with-session "env QT_QPA_PLATFORM=wayland WAYLAND_DISPLAY=ld-test QT_SCALE_FACTOR=1.25 ./lockdeader"
    ```
- `LOCKDEADER_DATA_DIR` overrides the data directory (default `~/.local/share/Lockdeader` on Linux), so it can point at a scratch directory during development.
- The first build is slow because miqt's Qt bindings and the vendored miniaudio compile through cgo; after that, builds are cached.

## Architecture

**Go ↔ QML bridge.** `main.go` creates Go-side objects and exposes them to QML as context properties (`app`, `mods`, `details`, `installed`, `installedMods`, `otherMods`, `otherModsModel`, `categories`, `search`, `clipboard`, `soundPreview`, `settings`, `uiState`, `launcher`, `retryAction`, `retryCategoriesAction`). Most of them are `QQmlPropertyMap`s rather than custom QObject subclasses: Go `Insert`s values that QML binds to, and QML asks Go to do something by *writing a key* (e.g. `details.id = 42`, `categories.selected = {section, id}`, `clipboard.copyImage = url`), which Go picks up in `OnValueChanged`. That callback only fires for writes from QML, not for Go's own `Insert`s. The pattern is documented in the type comments in `details.go` and `clipboard.go`; follow it for new bridges.

**Version.** `VERSION.txt` at the root holds the app's version (`MAJOR.MINOR.PATCH`), the one place it's set. `version.go` embeds it (`version`), and `main.go` passes it to the GameBanana client's user agent (`banana.UserAgent`). See `docs/distribution.md`.

**Packaging.** `packaging/` has the desktop file and AppStream metainfo (app ID `io.github.treejadey.Lockdeader`, which `main.go` also sets as the desktop file name), `packaging/flatpak/` a manifest that builds this checkout as a Flatpak, and `packaging/windows/` the script that makes the Windows app's folder with `windeployqt`, `ntldd` and a `qt.conf`, plus a smoke test that starts it on a clean machine (READMEs in each). `.github/workflows/build.yml` runs them on GitHub Actions and uploads the folder (which GitHub zips) and the Flatpak bundle (which it doesn't) as artifacts. The build scripts are PowerShell, so one language covers both platforms; both runners have it. Inside a Flatpak, `deploy.Running` asks the host's `pgrep` through `flatpak-spawn --host`, as Chatterino does. The plan for the rest is `docs/distribution.md`.

**Settings and state.** Settings are what the user chooses on the settings page (backed by `settingspage.go`) or by editing `Settings.toml`. App state the app just remembers, like the sound preview volume, goes in the database instead (see `appdb` below). `main.go` writes both half a second after the last change (`debounced`), and on quit. They're never saved over a file that failed to load, so a user's broken edit isn't lost.
- Where the user left the app (page, category, views, scroll positions, open mod) is `appdb.UIState`, kept up to date through the `uiState` bridge (`uistate.go`). On startup, `main.go` restores the category straight away. It then has the mod list `Seek` the top mod and the open mod in its first `restorePages` pages, and starts at the top if they aren't there. A page that saves its scroll position doesn't do so until it has restored it (`scrollRestored`), or the starting position would overwrite the saved one.
- On quit, if the session changed the file, what it held at startup is kept as a backup next to it (`Settings.toml.bkp-1` is the newest, up to `bkp-10`), like Chatterino's backups; see `appdata.BackUpSettings`.
- If the file fails to load and there are backups that would load, `qml/RestoreDialog.qml` asks before the main window opens whether to restore one (keeping the broken file as `Settings.toml.broken`), carry on with the defaults, or quit. It runs in its own engine and event loop (`restoredialog.go`), so `useNativeTextRendering` is called before settings are loaded.
- New settings go in `appdata.Settings` with a default in `DefaultSettings`, then get a key on the `settings` bridge and a row in `qml/SettingsPage.qml` (`SettingToggle` for on/off, `SettingRow` with any control). The page's search matches each row's text, description and `keywords`. A new `SettingsSection` shows up in the sidebar's list of sections (`SettingsNav`) by itself.

**Running and quitting.** Only one copy runs per data directory (`singleinstance.go`). Starting another asks the running one to show its window (`app.showRequests`), then exits. Closing the window while mods are installing only hides it: the installs finish, then the app quits (`main.go`, `installedMods.onIdle`). A desktop notification tells of any install that fails meanwhile (`onFailed`, `notification/`).

**Threading.** All Qt objects must be touched on the Qt main thread. Network work runs in a goroutine and hands its result back with `mainthread.Start(...)`. Because a response can arrive after the user has moved on, callbacks check that the result is still wanted (see `modDetails.Select`).

**Mod list model** (`modmodel.go`): a `QStandardItemModel` with custom role names, paged lazily through `canFetchMore`/`fetchMore` as the grid scrolls. It isn't a `QAbstractListModel` subclass because miqt v0.14.0's `roleNames` override frees its return value before Qt reads it.

**Installing mods** (`installed.go`): the install dialog writes the picked file ids to `details.install`, and `installedMods` queues the install (two run at once). `install.Fetch` downloads each archive, checks its MD5 and adds its VPKs to the store (each checked with `vpk.ReadDir`, so one the game can't load never gets in; an archive with no VPK but the game's compiled files loose in it is packed into one); only then is the mod added to the database, turned on. Rows of the installed mods model show an install's progress (`status`, `progress`, `pace`) until it's done, and how many options (`variants`) the mod has: its files on GameBanana, often versions of it, plus each VPK of an installed archive that has several (`configure.go`). With more than one, its right-click menu opens `ModConfigDialog` to pick which it uses (`installed.configure`, then `installed.setConfiguration`); picked files that aren't installed are downloaded first, and unpicked ones are removed only once they are. The page looks up the installed mods' files on GameBanana when it's shown (`installed.refresh`); `installedMods` is the model filtered by the page's search. On startup, `main.go` deletes leftover partial downloads, and VPKs no installed mod has (only when the real database opened).

**Putting mods into the game** (`deployer.go`, package `deploy`): the mods that are on go into `game/citadel/lockdeader/` as `pakNN_dir.vpk` files in load order, hard-linked from the store (copied if that fails), with a manifest saying what each slot holds; past 99 of them the lowest are bundled into generated VPKs. `installed.onChange` applies half a second after every change (`deployer.ApplySoon`), Play With Mods applies first, and startup applies to recover from a crash. Nothing is changed while Deadlock runs (`deploy.Running`); the change waits until it closes, which `gameWatcher` (`gamewatcher.go`) notices by checking every two seconds. Why it failed shows in the `InfoBar` on the installed page (`installed.applyError`, with Retry writing `applyNow`). Lockdeader's own lines in `gameinfo.gi` (`deploy.EnableMods`/`DisableMods`, marked `// Lockdeader`, with the addons line too when the "Also load mods installed by other tools" setting is on) are only there around its own launches: Play With Mods puts them in; Play Without Mods, startup and the game closing take them out (the last two as part of an apply, `deployer.ResetSoon`, so a launch that follows gets them back), so launching from Steam runs vanilla. `gameinfo.gi` is backed up in the data dir before the first write. See `docs/mod-support-plan.md`.

**Mods from other tools** (`othermods.go`, `deploy.ScanOtherMods`): VPKs in `game/citadel/addons` (and `addons2`…) that Deadlock Mod Manager's ledger (`.dmm.json`) lists, or that are just there, are shown in the installed page's "From Other Tools" section (the views' `footer`, with `readOnly` and `embedded` set), with names and pictures looked up on GameBanana by the ledger's ids (an id that isn't a Deadlock mod's is tried as a sound's). They can't be changed there; `TakeOverDialog` (from the page's info bar or Settings › Game) takes them over: a mod GameBanana knows is downloaded from there as a normal install (`installed.InstallThen`, with the dialog asking which file for mods that have several, guessed from the VPK's name), so it's no different from one installed here, and its files leave the addons folder, and the ledger, once the install succeeds. A file GameBanana doesn't know goes into the store as it is, as a local mod: section `Local`, with negative ids (see `appdb.Mod`).

**Translations.** gettext PO files in `po/`, as elementary OS apps use: `lockdeader.pot`, the template, made from the source by `cmd/translations`, and a `<language>.po` for each language (e.g. `de.po`, `pt_BR.po`), which translators make and fill in; never add or fill one in yourself. They're embedded in the binary (`translations.go`). On startup, `installTranslations` picks the first of the system's languages (`QLocale::uiLanguages`: `LANGUAGE`, then `LC_ALL`/`LC_MESSAGES`/`LANG` on Linux, the display language on Windows) that has a file, stopping at English, and installs a translator that answers every `qsTr` and `tr` from it (`i18n.Translator`), plus Qt's own translation of its own text into the same language. It's installed for English too, which needs it for plurals. There's no setting for the language. `po/README.md` is for translators.

**C++ shims via cgo.** Where miqt doesn't bind something, a small C++ file with a C header and a Go wrapper fills the gap: `imagecache.{cpp,h,go}` (an on-disk network cache for QML images, which miqt can't do because it doesn't bind `QQmlNetworkAccessManagerFactory::create`) `nativetext.{cpp,h,go}` (native text rendering, since miqt has no Qt Quick bindings) and `translator.{cpp,h}` with `translations.go` (a `QTranslator` that asks Go, since miqt v0.14.0 frees the string a `translate` override returns before Qt reads it). All must be called before any QML is loaded. The clipboard bridge reads the same disk cache, so it can copy images that are already on screen without downloading them again.

**miqt functions to avoid.** In miqt v0.14.0, setters that take a list of pairs allocate 8 bytes per pair but write 24, which corrupts the heap: `QNetworkCacheMetaData.SetRawHeaders`, `QGradient.SetStops`, `QVariantAnimation.SetKeyValues` and `QUrlQuery.SetQueryItems`. glibc doesn't notice, so tests pass on Linux, but on Windows the process dies later, somewhere else, with `0xC0000374` and no message. Don't call them (`TestCopyImageFromCache` fills the cache through a real request instead), or fill the gap with a C++ shim. To find a crash like this, build the tests with AddressSanitizer, which stops at the bad write: `CC=clang CXX=clang++ go test -asan -c` (clang if gcc's `libasan` isn't installed), then run the binary with `ASAN_OPTIONS=detect_leaks=0`.

**Packages.**
- `banana/`: a minimal client for the GameBanana v11 API, with no Qt dependency. Tests use `httptest` servers and the JSON fixtures in `banana/testdata/`.
- `audio/`: plays sounds through a vendored copy of [miniaudio](https://miniaud.io) (with stb_vorbis for Ogg), wrapped in a small C layer (`audio.c`/`audio.h`). See `audio/miniaudio/README.md`.
  - Don't use Qt Multimedia: it plays through FFmpeg.
  - Tests use `audio.NewEngineWithoutDevice`, which mixes only when `Read` is called, so they need no sound card. The `soundPreview` bridge (`soundpreview.go`) takes one in its tests too.
- `vpk/`: reads a VPK's directory (`ReadDir`: paths, sizes, CRCs, whether data is in a chunk file) and writes VPKs by streaming the files given to it (`Write`). No deps; tests build VPKs in code.
- `deploy/`: puts mods into the game (`Apply`, with the slot allocator in `alloc.go`), edits `gameinfo.gi` (`EnableMods`, `DisableMods`, `ReadGameInfo`), lists other tools' mods in the addons folders (`ScanOtherMods`, `ForgetInLedger` for DMM's `.dmm.json`) and tells whether Deadlock is running (`Running`, per platform). No Qt dependency. Tests use fake game folders under `t.TempDir()`; `testdata/gameinfo.gi` is the stock file's head, CRLF and all.
- `install/`: downloads mods (`Fetch`), takes the VPKs out of zip, 7z and rar archives (`Extract`), and keeps them in the store, `Cache/Mods`, named by SHA-256 (`Store`). `SpeedMeter` sorts a download's speed into a pace for its progress bar: slow under ~17 Mbps, fast over ~60 Mbps. No Qt dependency; `install/testdata/mod.7z` is a test fixture, and the rar tests build their archive in code.
- `i18n/`: reads and writes PO files (`Parse`, `File.Write`, which lays them out as gettext's tools do, so a file saved in Gtranslator or Poedit round-trips unchanged; `TestWriteMatchesMsgcat` checks that where `msgcat` is installed), extracts the text to translate (`Extract`: `qsTr` in `qml/*.qml` and `qml/*.js`, `tr`/`trN` in the root package's Go files), updates a language's file from the template (`Update`, like `msgmerge` without its guesses), checks one (`Check`: plural forms, placeholders), and translates at run time (`Load`, `Match`, `Translator`). No Qt dependency. `cmd/translations` is its command line.
- `rcc/`: writes Qt's binary resource format (what `rcc --binary` makes) from an `fs.FS`, so embedded files can be handed to Qt without Qt's `rcc` at build time. No Qt dependency.
- `appdata/`: locates the per-platform data directory and its layout (`Lockdeader.db`, `Settings/Settings.toml`, `Cache/Images`, `Cache/Mods`), and loads TOML settings, falling back to defaults for missing keys.
- `game/`: starts Deadlock through Steam (`Launch`: Steam's executable with `-applaunch 1422450` and the launch options as separate arguments, split by `SplitOptions`; `SteamURL` when the executable can't be found), finds Deadlock in Steam's libraries (`Find`: the usual Steam folders on Linux, the registry and Program Files on Windows, then `libraryfolders.vdf` and the app manifest), and checks that a folder is Deadlock's (`Check`: it has `game/citadel/gameinfo.gi`; a folder inside the game counts as the game's). No Qt dependency.
  - The game folder setting is only ever "" or a folder `Check` accepts. A folder the user picks that isn't Deadlock's is turned down (`settings.gameFolderError`). One that's stopped being Deadlock's is dropped on startup, and the game is looked for again, also whenever the settings page is shown without one.
- `filedialog/`: the system's own folder dialog, opened from Go (`ChooseFolder`, in a goroutine). On Linux it goes through xdg-desktop-portal (`rymdport/portal`), falling back to `zenity`/`kdialog`; on Windows it's Windows' own folder picker (`ncruces/zenity`).
  - Don't use Qt Quick Dialogs: without a QtWidgets platform theme they fall back to Qt's own QML dialog, which looks nothing like the desktop's.
- `notification/`: desktop notifications, through D-Bus (`godbus/dbus`) on Linux and Windows toasts (`go-toast`) on Windows. On Windows, the first one registers the app under the user's registry key, as Windows requires.
- `appdb/`: the SQLite database, through the pure-Go `modernc.org/sqlite`. It holds app state (`app_state`, a key/value table, with typed accessors like `AudioState`) and installed mods (`mods`, `mod_files`, `mod_vpks`). A mod or file that isn't from GameBanana is added with id 0 and gets minus its own row id, so the unique constraints hold without a schema change.
  - **Schema versions:** the version is SQLite's `user_version`. To change the schema, append to `migrations` in `appdb/schema.go`; never edit a released migration. Each migration runs in a transaction along with its version bump.
  - **Never written over:** a database from a newer version (`ErrNewerSchema`) or one that won't open is left alone, and the session runs on `OpenMemory` instead.
  - Tables are `STRICT`, and times are Unix seconds.
- `package main` tests (`clipboard_test.go`) need a live Qt app. `TestMain` runs Qt's event loop on the locked main thread with the offscreen platform, and tests hand their Qt calls over through `mainthread.Wait`/`Start`. Reuse that harness for new Qt-dependent tests.

## Code style

Applies to Go, QML (including its JavaScript), C and C++.

**Comments**
- Don't write a comment that says what a function, type or line does when its name and code already make that clear.
- Do comment:
  - code that may look unusual, surprising or wrong;
  - workarounds, such as for miqt or Qt bugs;
  - the reason behind a non-obvious choice.
- Put a short comment above a group of statements to say what it's for, when that isn't obvious from the code.

**Blank lines**
- Leave a blank line before and after every block (`if`, `for`, `switch`, `func` literal, QML child object, etc.). This doesn't apply at the start or end of the enclosing block: never leave a blank line right after `{` or right before `}`.
  - Exception: a check stays attached to the line that set what it checks: `err`, `ok`, nil, or the value itself (`dir := getenv(...)` then `if dir == ""`).

    ```go
    data, err := os.ReadFile(path)
    if err != nil {
    	return err
    }
    ```
- Group statements that belong together, and separate the groups with blank lines, e.g. setting up an object, then wiring its callbacks, then using it. Steps that do different things get their own group, even inside a short block:

  ```go
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
  ```
- Leave a blank line before a `return`, unless it's the first statement in its block (e.g. `if err != nil { return err }`). The same goes for `break` and `continue` that end a longer block.

## QML conventions

- Interface design and UI text follow [docs/interface-guidelines.md](docs/interface-guidelines.md), the parts of the elementary OS HIG that apply here. Read it before designing a view or dialog or writing UI text. In short:
  - Title Case for buttons, menu items, titles and placeholders; sentence case for labels, descriptions, tooltips and messages.
  - Every user-visible string is in `qsTr()`, or `tr()` (`tr.go`) for text Go hands to the UI.
  - Offer Undo rather than asking "Are you sure?".
  - Name actions, never "OK".
- All QML lives in `qml/`. New components must be registered in `qml/qmldir`; singletons are declared there with `singleton`. Windows (`Main.qml`, `RestoreDialog.qml`) are loaded by URL instead (`qmlURL`), and are `AppWindow`s, which carry the theme's palette.
- `qml/` is embedded in the binary (`qmlfiles.go`): `rcc.Build` packs it in Qt's resource format, and `registerQML` registers it, so QML loads from `qrc:/qml/`. Anything QML refers to must be inside `qml/`.
- Import `QtQuick.Controls.Fusion`, never plain `QtQuick.Controls`; otherwise KDE Plasma substitutes its own style and ignores the app theme.
- Don't name a component after a Qt Quick Controls type (e.g. Qt 6.10's `SearchField`): the control wins over the file in `qml/`. Hence `SearchBox`.
- Colours come from the `Theme` singleton: Rosé Pine while the system prefers a dark style, Rosé Pine Dawn otherwise, as elementary OS asks of apps that work in both. It follows the system's preference (`Application.styleHints.colorScheme`) and changes along with it, on Windows too (Qt reads the "default app mode" setting and watches for changes). Where the system states none (Windows' high-contrast themes, a Linux desktop without a settings portal), the system palette decides: a window darker than its text means dark. Anything new has to work in both:
  - pick colours by role rather than for how they look in one style: `accent`/`textOnAccent` for the main action (foam, with dark text, in both), `field` behind text being typed, `textOnPicture` for text and icons over pictures, `backdrop` (dark in both styles) behind dialogs and the image viewer, with `textOnBackdrop` for text drawn straight on it;
  - for hovering, use `Theme.lift()` rather than `Qt.lighter`, which barely shows in the light style.
- Animation durations and easing curves come from the `Motion` singleton (IBM Carbon motion tokens, productive and expressive). Don't hard-code either.
- Fonts are bundled in `fonts/` (see `fonts/README.md`): Inter for all UI text, JetBrains Mono for monospace. `fonts.go` embeds and loads them.
  - **Inter:** set as the app's default font, at the platform's UI font size. The bundled files have the `ss04` stylistic set (a tailed l, a serifed I) built in, and are renamed `Inter SS04` so an installed Inter never stands in for them.
  - **Monospace in QML:** use `font.family: app.monospaceFont`; never rely on the system's monospace font.
  - **Monospace in descriptions:** `codeStyle()` sets `<code>`/`<pre>` in rich text descriptions to JetBrains Mono.
  - **Explicit `font.pixelSize` values** are chosen for Inter.
- Icons are Microsoft Fluent UI System Icons SVGs in `qml/icons/`. Add new icons from that set, keeping the original path data and setting the fill colour on the `<svg>` element; don't draw your own.
  - That fill is the dark style's text colour, so on the app's own surfaces draw icons with `TintedIcon` (which defaults to `Theme.text`), or `icon.color` on a Qt Quick Controls item such as a menu item. A plain `Image` is only for icons over pictures, which stay light.
- `qml/icons/bitmap/` holds the app's own pictures, which aren't icons, e.g. `sound_icon.png`, shown on the card of a sound, which has no picture on GameBanana. Don't put GameBanana's own artwork in the app: it's theirs to distribute.
- Icons are outlined (`*_24_regular.svg`) while their element is inactive, and switch to the filled version (`*_24_filled.svg`) while it's active. Examples:
  - the current page in the sidebar (`SidebarItem` swaps `_regular` for `_filled` itself, so both files must exist);
  - play turning into a filled pause while playing;
  - the volume button while its slider is open.

  Icons that never have an active state (static or one-shot actions) only need the regular version.
- Don't put an icon on every button. Use one only where it has a clear purpose, such as helping users recognise a frequently used control at a glance (e.g. the main navigation, Back, Settings). Lists of options, such as the category list, are plain text.
- Text has to stay sharp at fractional display scaling (e.g. 125%): use `TextShadow` (a sibling drawn under the text) rather than `MultiEffect`'s own shadow on the text, which blurs it.
- `SmoothWheel` gives a `Flickable` smooth mouse-wheel scrolling; put it over any new scrolling view.
- Dialogs are built on `AppDialog` (the frame: backdrop, card, title with an optional dimmed start, close button, buttons in the bottom-right corner, and its motion), e.g. `ModConfigDialog`. The card is as tall as its content; when that's more than the window, the card scrolls as a whole over the fixed backdrop, so content that grows (like `TakeOverDialog`'s lists) shouldn't scroll inside. `InstallDialog` predates it and still has its own frame.
- Disabled controls all look the same: a slight gradient from brighter at the top to darker at the bottom, with muted text. Lay a `DisabledFill` over the control's background (`mark: true` for small marks such as a checkbox), and make that background clear while disabled, or its antialiased edge shows as a thin outline; the colours are `Theme.disabled*`. Don't fade disabled controls with opacity.
- Errors reach QML in two keys (or roles): `error`, in plain words from `errorMessage` (`errormessage.go`), and `errorDetail`, the error as Go wrote it; `insertError` sets both on a bridge. Show them with `ErrorLabel`, which has the detail on hover, after a line saying what failed ("Couldn’t load mods"). A new kind of failure gets a case in `errorMessage`, so check that its error can be told apart with `errors.Is`/`As`.
- Short confirmations are `Toast`s. The app-wide one, at the top of the window over any dialog, is in `Main.qml` and shows each `installed.message` (bridges announce one by bumping a `messageId`, as `clipboard` does).
  - A toast can carry one action. Uninstalling offers Undo this way (`messageUndoable`, with Ctrl+Z too). The mod stays in the database until the toast has gone (`messageGone`) or the app quits (`FinishUninstall`).
- For an icon whose colour changes with state, use `TintedIcon`, which colours the SVG's shape, rather than an SVG per colour.
- Right-click menus are `PopupMenu`s of `PopupMenuItem`s. Hide an item that doesn't apply with `shown: false`, never with `visible`/`height` bindings: `visible` is false while the menu is closed, so those loop.
- Lists of a mod's files to pick from are `FilePickerList`, shared by `InstallDialog` and `ModConfigDialog`; change how picking files looks there, not in a dialog.
- The sidebar's Play With Mods and Play Without Mods buttons write `launcher.play`. Both are disabled, not hidden, until the game folder is known, and Play With Mods until a mod is installed too (`installed.installedCount`); pressed without a folder, either still answers, with a toast whose Settings action opens the settings (`launcher.messageAction`). Both are disabled while Deadlock runs (`launcher.gameRunning`), with a tooltip saying so.
- Notices that stay until dealt with are `InfoBar`s, in the view under the top bar (e.g. the installed page's, for mods not reaching the game), with a severity, the detail on hover and one action.
- `ActionButton`'s `sheen` shades it from nearly white at the top down to its colour, for the one action that should stand out most, like Play With Mods.
- Checkboxes are `TickBox` (Carbon-style; clicking emits `clicked()` without changing `checked`, so the model decides), and progress bars are `DownloadProgress` (elementary OS-style, with a shine whose `pace` follows the download speed). `InstallStatus` pairs the bar with its label.
