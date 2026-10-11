# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Lockdeader is a desktop mod manager for the game Deadlock (still a work in progress: it browses mods on GameBanana and downloads them into its own store, but doesn't put them in the game folder yet). It is written in Go with a Qt 6 Quick/QML UI, using the [miqt](https://github.com/mappu/miqt) bindings.

## Commands

```sh
go build -o lockdeader .     # needs Qt 6 dev packages (pkg-config: Qt6Qml, Qt6Network, Qt6Quick) and a C++ compiler; uses cgo
./lockdeader                 # run from the repo root: main.go loads qml/Main.qml by relative path
go test ./...                # all tests
go test ./banana -run TestListModsDecode   # a single test
go vet ./... && gofmt -l .
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

**Go ↔ QML bridge.** `main.go` creates Go-side objects and exposes them to QML as context properties (`app`, `mods`, `details`, `installed`, `installedMods`, `categories`, `search`, `clipboard`, `soundPreview`, `settings`, `retryAction`, `retryCategoriesAction`). Most of them are `QQmlPropertyMap`s rather than custom QObject subclasses: Go `Insert`s values that QML binds to, and QML asks Go to do something by *writing a key* (e.g. `details.id = 42`, `categories.selected = {section, id}`, `clipboard.copyImage = url`), which Go picks up in `OnValueChanged`. That callback only fires for writes from QML, not for Go's own `Insert`s. The pattern is documented in the type comments in `details.go` and `clipboard.go`; follow it for new bridges.

**Settings and state.** Settings are what the user chooses on the settings page (backed by `settingspage.go`) or by editing `Settings.toml`. App state the app just remembers, like the sound preview volume, goes in the database instead (see `appdb` below). `main.go` writes both half a second after the last change (`debounced`), and on quit. They're never saved over a file that failed to load, so a user's broken edit isn't lost.
- On quit, if the session changed the file, what it held at startup is kept as a backup next to it (`Settings.toml.bkp-1` is the newest, up to `bkp-10`), like Chatterino's backups; see `appdata.BackUpSettings`.
- If the file fails to load and there are backups that would load, `qml/RestoreDialog.qml` asks before the main window opens whether to restore one (keeping the broken file as `Settings.toml.broken`), carry on with the defaults, or quit. It runs in its own engine and event loop (`restoredialog.go`), so `useNativeTextRendering` is called before settings are loaded.
- New settings go in `appdata.Settings` with a default in `DefaultSettings`, then get a key on the `settings` bridge and a row in `qml/SettingsPage.qml` (`SettingToggle` for on/off, `SettingRow` with any control). The page's search matches each row's text, description and `keywords`. A new `SettingsSection` shows up in the sidebar's list of sections (`SettingsNav`) by itself.

**Threading.** All Qt objects must be touched on the Qt main thread. Network work runs in a goroutine and hands its result back with `mainthread.Start(...)`. Because a response can arrive after the user has moved on, callbacks check that the result is still wanted (see `modDetails.Select`).

**Mod list model** (`modmodel.go`): a `QStandardItemModel` with custom role names, paged lazily through `canFetchMore`/`fetchMore` as the grid scrolls. It isn't a `QAbstractListModel` subclass because miqt v0.14.0's `roleNames` override frees its return value before Qt reads it.

**Installing mods** (`installed.go`): the install dialog writes the picked file ids to `details.install`, and `installedMods` queues the install (two run at once). `install.Fetch` downloads each archive, checks its MD5 and adds its VPKs to the store; only then is the mod added to the database, turned on. Rows of the installed mods model show an install's progress (`status`, `progress`, `pace`) until it's done, and how many options (`variants`) the mod has: its files on GameBanana, often versions of it, plus each VPK of an installed archive that has several (`configure.go`). With more than one, its right-click menu opens `ModConfigDialog` to pick which it uses (`installed.configure`, then `installed.setConfiguration`); picked files that aren't installed are downloaded first, and unpicked ones are removed only once they are. The page looks up the installed mods' files on GameBanana when it's shown (`installed.refresh`); `installedMods` is the model filtered by the page's search. On startup, `main.go` deletes leftover partial downloads, and VPKs no installed mod has (only when the real database opened).

**C++ shims via cgo.** Where miqt doesn't bind something, a small C++ file with a C header and a Go wrapper fills the gap: `imagecache.{cpp,h,go}` (an on-disk network cache for QML images, which miqt can't do because it doesn't bind `QQmlNetworkAccessManagerFactory::create`) and `nativetext.{cpp,h,go}` (native text rendering, since miqt has no Qt Quick bindings). Both must be called before any QML is loaded. The clipboard bridge reads the same disk cache, so it can copy images that are already on screen without downloading them again.

**Packages.**
- `banana/`: a minimal client for the GameBanana v11 API, with no Qt dependency. Tests use `httptest` servers and the JSON fixtures in `banana/testdata/`.
- `audio/`: plays sounds through a vendored copy of [miniaudio](https://miniaud.io) (with stb_vorbis for Ogg), wrapped in a small C layer (`audio.c`/`audio.h`). See `audio/miniaudio/README.md`.
  - Don't use Qt Multimedia: it plays through FFmpeg.
  - Tests use `audio.NewEngineWithoutDevice`, which mixes only when `Read` is called, so they need no sound card. The `soundPreview` bridge (`soundpreview.go`) takes one in its tests too.
- `install/`: downloads mods (`Fetch`), takes the VPKs out of zip, 7z and rar archives (`Extract`), and keeps them in the store, `Cache/Mods`, named by SHA-256 (`Store`). `SpeedMeter` sorts a download's speed into a pace for its progress bar: slow under ~17 Mbps, fast over ~60 Mbps. No Qt dependency; `install/testdata/mod.7z` is a test fixture, and the rar tests build their archive in code.
- `appdata/`: locates the per-platform data directory and its layout (`Lockdeader.db`, `Settings/Settings.toml`, `Cache/Images`, `Cache/Mods`), and loads TOML settings, falling back to defaults for missing keys.
- `appdb/`: the SQLite database, through the pure-Go `modernc.org/sqlite`. It holds app state (`app_state`, a key/value table, with typed accessors like `AudioState`) and installed mods (`mods`, `mod_files`, `mod_vpks`).
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

- All QML lives in `qml/`. New components must be registered in `qml/qmldir`; singletons are declared there with `singleton`. Windows (`Main.qml`, `RestoreDialog.qml`) are loaded by path instead, and are `AppWindow`s, which carry the theme's palette.
- Import `QtQuick.Controls.Fusion`, never plain `QtQuick.Controls`; otherwise KDE Plasma substitutes its own style and ignores the app theme.
- Don't name a component after a Qt Quick Controls type (e.g. Qt 6.10's `SearchField`): the control wins over the file in `qml/`. Hence `SearchBox`.
- Colours come from the `Theme` singleton (the Rosé Pine palette). Animation durations and easing curves come from the `Motion` singleton (IBM Carbon motion tokens, productive and expressive). Don't hard-code either.
- Fonts are bundled in `fonts/` (see `fonts/README.md`): Inter for all UI text, JetBrains Mono for monospace. `fonts.go` embeds and loads them.
  - **Inter:** set as the app's default font, at the platform's UI font size. The bundled files have the `ss04` stylistic set (a tailed l, a serifed I) built in, and are renamed `Inter SS04` so an installed Inter never stands in for them.
  - **Monospace in QML:** use `font.family: app.monospaceFont`; never rely on the system's monospace font.
  - **Monospace in descriptions:** `codeStyle()` sets `<code>`/`<pre>` in rich text descriptions to JetBrains Mono.
  - **Explicit `font.pixelSize` values** are chosen for Inter.
- Icons are Microsoft Fluent UI System Icons SVGs in `qml/icons/`. Add new icons from that set, keeping the original path data and setting the fill colour on the `<svg>` element; don't draw your own.
- Icons are outlined (`*_24_regular.svg`) while their element is inactive, and switch to the filled version (`*_24_filled.svg`) while it's active. Examples:
  - the current page in the sidebar (`SidebarItem` swaps `_regular` for `_filled` itself, so both files must exist);
  - play turning into a filled pause while playing;
  - the volume button while its slider is open.

  Icons that never have an active state (static or one-shot actions) only need the regular version.
- Don't put an icon on every button. Use one only where it has a clear purpose, such as helping users recognise a frequently used control at a glance (e.g. the main navigation, Back, Settings). Lists of options, such as the category list, are plain text.
- Text has to stay sharp at fractional display scaling (e.g. 125%):
  - Put `transform: PixelNudge {}` on containers that hold text.
  - Use `TextShadow` (a sibling drawn under the text) rather than `MultiEffect`'s own shadow on the text, which blurs it.
- `SmoothWheel` gives a `Flickable` smooth mouse-wheel scrolling; put it over any new scrolling view.
- Dialogs are built on `AppDialog` (the frame: backdrop, card, title with an optional dimmed start, close button, buttons in the bottom-right corner, and its motion), e.g. `ModConfigDialog`. `InstallDialog` predates it and still has its own frame.
- Disabled controls all look the same: a slight gradient from brighter at the top to darker at the bottom, with muted text. Lay a `DisabledFill` over the control's background (`mark: true` for small marks such as a checkbox), and make that background clear while disabled, or its antialiased edge shows as a thin outline; the colours are `Theme.disabled*`. Don't fade disabled controls with opacity.
- Short confirmations are `Toast`s. The app-wide one, at the top of the window over any dialog, is in `Main.qml` and shows each `installed.message` (bridges announce one by bumping a `messageId`, as `clipboard` does).
- For an icon whose colour changes with state, use `TintedIcon`, which colours the SVG's shape, rather than an SVG per colour.
- Checkboxes are `TickBox` (Carbon-style; clicking emits `clicked()` without changing `checked`, so the model decides), and progress bars are `DownloadProgress` (elementary OS-style, with a shine whose `pace` follows the download speed). `InstallStatus` pairs the bar with its label.
