# Plan: distributing Lockdeader

How Lockdeader should reach people (2026-10-12; decisions recorded the same day). Nothing here is built yet.

- **Windows:** a `.zip` with everything the app needs to run, built by GitHub Actions.
- **Linux:** packages built on openSUSE's Open Build Service (OBS), and a Flatpak on Flathub later.

The Linux channels depend on others accepting us: Flathub's reviewers, and openSUSE's for packages that go into the distribution itself. The plan doesn't count on either. Until then, there's OBS's home project, which needs nobody's approval.

## Principles

- **Same app everywhere.** Every channel ships the same source with the same features. Where a channel needs the app to behave differently (e.g. the Flatpak sandbox), the app works it out at run time; the channels don't patch it.
- **Every channel builds from a tagged release.** CI runs on every push to catch breakage early, but what users download always comes from a tag.
- **Licences travel with the app.** Lockdeader is GPL-3.0. What we bundle comes with its own licences, which every package must include:
  - Qt: LGPL-3.0, dynamically linked so users can swap it;
  - the fonts: OFL;
  - the Fluent icons: MIT;
  - miniaudio: public domain or MIT-0.

## Before any of it: what the app needs

These are needed by all three channels, so they come first.

### Finding its QML

`main.go` and `restoredialog.go` load `qml/Main.qml` and `qml/RestoreDialog.qml` by a path relative to the **working directory**. That only works when running from the repository's root. A zip can be started from anywhere, and a package puts the QML somewhere like `/usr/share`.

**Built: the QML is embedded in the binary**, as the fonts and translations already are.

- `go:embed` bundles `qml/` (`qmlfiles.go`).
- At startup, the `rcc` package packs it into Qt's binary resource format, and `QResource::registerResource` hands it to Qt.
- QML loads from `qrc:/qml/Main.qml`.
- Writing the format in Go means there's no build step and no need for Qt's `rcc` tool: `go build` is still all it takes.
- The result is one file, with nothing to install and nothing for a channel to put in the right place.
- QML changes now need a rebuild. If that gets in the way, an environment variable could point at `qml/` on disk instead.

Looking for `qml/` next to the executable, then in the install prefix, was the alternative. It was turned down because every channel would have to install the files in the right place.

### A version

**Built: the version is in `VERSION.txt`** at the repository's root (`MAJOR.MINOR.PATCH`). `version.go` embeds it, so nothing has to be passed at build time.

- **Where the app uses it:** in the user agent it sends to GameBanana, and in the log at startup.
- **To release:** change `VERSION.txt` and tag the commit `v` plus that version. CI can check that the two agree.
- **Packaging:** manifests and spec files are updated by hand for each release, and can read the version from `VERSION.txt` where their tools allow.
- **Still to do:** showing it on the settings page, and in AppStream's release notes.

### Desktop integration files

**Built in `packaging/`:** the desktop file, the metainfo (summary, description, screenshot, OARS, `GPL-3.0-or-later`), and the scalable icon (`assets/logo.svg`, installed under the app ID). The window sets its desktop file name. Still to do: the PNG sizes, the Windows `.ico` and `.syso`, and release notes. What each needs:

- **App ID: `io.github.treejadey.Lockdeader`.** Flathub verifies it through the GitHub account. It names the desktop file, metainfo, icons and Flatpak.
- **`<app-id>.desktop`:** name, icon, categories (`Game;Utility;`), and `StartupWMClass` so the window groups with its launcher.
- **`<app-id>.metainfo.xml`:** AppStream metadata.
  - summary and description;
  - screenshots (`Images/app_screenshot.png` is a start);
  - release notes for each version;
  - OARS content rating;
  - project and metadata licences;
  - links to the project and to where bugs go.

  Flathub requires it, software centres show it for OBS packages, and `appstreamcli validate` checks it in CI.
- **Icons:** the app icon as SVG (`assets/logo.svg`), plus PNGs at the sizes the interface guidelines ask for (16 to 128 px, each drawn for its size). Windows gets an `.ico` built from the same, embedded in the `.exe` with a `.syso` resource (`goversioninfo`, which also sets the file's version and description).

The window should also set its desktop file name (`QGuiApplication::setDesktopFileName`), so Wayland compositors match it to its icon.

### Third-party notices

A `THIRD-PARTY-NOTICES` file (or a folder of licences) listing what's bundled, with the licence texts. Every channel ships it.

## Windows: a zip from GitHub Actions

### What's in it

```
Lockdeader-1.2.3-windows-x64.zip
├── Lockdeader.exe          built with -H windowsgui, so no console window
├── Qt6*.dll                Core, Gui, Network, Qml, Quick, QuickControls2, ...
├── lib*.dll                the MinGW runtime and Qt's own dependencies
├── platforms/qwindows.dll
├── imageformats/           qsvg, qjpeg, qgif, qwebp: icons and GameBanana's pictures
├── iconengines/qsvgicon.dll
├── tls/                    HTTPS for images loaded through Qt (qschannelbackend)
├── qml/                    Qt's QML modules: QtQuick, Controls (Fusion), Layouts, Effects
│                           (the app's own QML is in the .exe)
├── translations/           Qt's own (qt_*.qm), which the app loads for Qt's text
├── qt.conf                 where those are; MSYS2's Qt looks in share\qt6\ otherwise
├── LICENSE
└── THIRD-PARTY-NOTICES
```

Unzip and run. It's portable, apart from the data in `%LOCALAPPDATA%\Lockdeader`, which `appdata` already puts there. Toast notifications register the app under the user's registry key the first time (`notification_windows.go`), which a portable app can do.

### Building it

**Built:** `packaging/windows/build.ps1` and `smoke-test.ps1`, run by the `windows` and `windows-smoke-test` jobs of `.github/workflows/build.yml`, which upload the app's folder as an artifact (GitHub zips it, so the build doesn't). Attaching it to a release on a tag is still to do. The build scripts are PowerShell on both platforms, since both runners ship it. What follows is the reasoning.

cgo can't use MSVC, so Qt has to be a MinGW build. Which one doesn't matter much; use whichever is more convenient. That's likely **MSYS2's UCRT64 environment**:

- `msys2/setup-msys2` installs it on GitHub's `windows-latest` runner;
- its package manager provides Qt 6 and GCC;
- miqt is tested with it.

The workflow:

1. **Set up.** Install the MSYS2 packages: `mingw-w64-ucrt-x86_64-{gcc,pkgconf,qt6-base,qt6-declarative,qt6-svg,qt6-imageformats}`. Install the Go version from `go.mod` with `actions/setup-go`, with MSYS2's `gcc` and `pkg-config` on the `PATH`.
2. **Test.** `go test ./...` and `go vet ./...`. The `package main` tests use Qt's offscreen platform, which Windows has too.
3. **Build.**

   ```sh
   CGO_ENABLED=1 go build -ldflags "-s -w -H windowsgui" -o Lockdeader.exe .
   ```

4. **Collect the DLLs.**
   - `windeployqt --qmldir qml --release Lockdeader.exe` copies Qt's DLLs, plugins and the QML modules the app imports. MSYS2 may name it `windeployqt6`; check.
   - `windeployqt` doesn't copy what Qt itself links against from MSYS2: ICU, zstd, HarfBuzz, FreeType, libpng, PCRE2, and the MinGW runtime (`libgcc_s_seh-1`, `libstdc++-6`, `libwinpthread-1`). Find those with `ntldd -R` (MSYS2's `mingw-w64-ucrt-x86_64-ntldd`) and copy each one that lives in `/ucrt64/bin`.
5. **Smoke-test.** In a later job, on a clean runner without MSYS2, download the artifact and start the app with `QT_QPA_PLATFORM=offscreen` and `QT_FORCE_STDERR_LOGGING=1`. Fail if it exits early or logs a missing module or DLL. This is what catches a DLL left out.
6. **Publish.**
   - Every push: upload the folder as a workflow artifact, for trying out.
   - A `v*` tag: zip it and attach it to a GitHub Release, with a `.sha256` file.

MSYS2's Qt is built with ICU, which adds about 30 MB. If MSYS2 turns out awkward, Qt's official MinGW build is the fallback: install it with `aqtinstall`, along with its matching MinGW. It's built without ICU, so the zip is smaller, but there's more to set up.

### Later

- **Code signing.** Without it, SmartScreen warns about the `.exe` until it has a reputation. The SignPath Foundation signs open-source projects for free.
- **Package managers.** winget and Scoop manifests can point at the release zip.

## Linux: openSUSE's Open Build Service

OBS builds packages from a spec file in a clean chroot, for openSUSE and for other distributions too (Fedora, Arch, Debian/Ubuntu), and hosts repositories users can add.

### Steps

1. **A home project**, `home:<user>:lockdeader` or just `home:<user>`, needs no approval. It gives us Tumbleweed packages and a repository straight away.
2. **More distributions** go in the same project, as far as their Qt is new enough. The app is developed against Qt 6.11; the oldest Qt it builds and runs with still has to be found, by building against older ones. That decides which releases (Leap, Fedora's, Ubuntu's) can be built at all.
3. **openSUSE itself.** Submit the package to a fitting devel project, then from there to `openSUSE:Factory`, which feeds Tumbleweed. Both are reviewed, and the package then needs a maintainer who keeps it building.

### The package

- **Source.**
  - The release tag's tarball, fetched by an `obs_scm` service.
  - A `vendor.tar.gz` of the Go modules, made by `obs-service-go_modules`. OBS builds without the network, so Go can't download them.
  - Both are regenerated by `osc service runall` for each release.
- **`lockdeader.spec`.**
  - `BuildRequires`: a Go new enough for `go.mod`, `gcc-c++`, and `pkgconfig(Qt6Qml)`, `pkgconfig(Qt6Quick)`, `pkgconfig(Qt6Network)`.
  - `%build`: `go build -mod=vendor -buildmode=pie`. The version comes from `VERSION.txt`, which the spec's `Version:` should match.
  - `%check`: `go test ./...` (the Qt tests need `QT_QPA_PLATFORM=offscreen`).
  - `%install`: the binary into `%{_bindir}`, plus the desktop file, icons, metainfo and licences. If the QML is embedded, that's all.
- **Run-time dependencies.**
  - RPM finds the shared libraries by itself.
  - It doesn't find the QML modules the app imports. List those as `Requires`: QtQuick, Controls, Layouts and Effects; in openSUSE, packages like `qt6-declarative-imports` (check the names).
  - Same for the SVG image format plugin, which draws the icons.
- **Releasing.** A GitHub Actions job (or OBS's own SCM integration, with a token from `osc token`) asks OBS to rebuild when a tag is pushed.

The same spec can cover Fedora with a few conditionals. Debian and Ubuntu need their own packaging (`debian/` or a `.dsc`); add them only once someone asks.

## Linux: Flatpak on Flathub

### The manifest

**Built: `packaging/flatpak/io.github.treejadey.Lockdeader.yml`** builds this checkout for trying out (see its README). Two things in it aren't ready for Flathub:

- it lets Go download the modules during the build;
- Steam libraries on other drives aren't reachable.

- **Runtime:** `org.kde.Platform` and `org.kde.Sdk`, on the branch with the Qt we need. They provide Qt 6 with Qt Quick, so we don't bundle Qt.
- **Go:** `org.freedesktop.Sdk.Extension.golang`. Flathub builds without the network, so the Go modules are listed as sources, generated with `flatpak-builder-tools`' Go module tool, or come vendored in a release tarball.
- **Build:** `go build` as for OBS, then install the binary, desktop file, icons and metainfo into `/app`.

### What the sandbox changes

| What the app does | In the Flatpak | What's needed |
|---|---|---|
| Finds Deadlock in Steam's libraries (`game.Find`, `steamRoots`) and writes to its folder (`deploy.Apply`, `gameinfo.gi`) | Can't see the home folder | Static permissions for where Steam keeps games: `--filesystem=xdg-data/Steam`, `--filesystem=~/.steam`, and `~/.var/app/com.valvesoftware.Steam` for Steam's Flatpak. Libraries on other drives need `/mnt`, `/run/media` and so on, or a folder the user picks |
| The user picks the game folder (`filedialog`, through the portal) | Works, but a folder without a static permission arrives as a document portal path (`/run/user/…/doc/…`) | Hard links from the store fail there, and `deploy` already falls back to copying. Check `Check`, `ReadGameInfo` and the addons scan through such a path |
| Tells whether Deadlock is running (`deploy.Running`, through `/proc`) | **Can't see the game:** the sandbox has its own process namespace | Ask the host instead, as Chatterino does (see below), with `--talk-name=org.freedesktop.Flatpak` |
| Starts Deadlock (`game.Launch`) | No `steam` command in the sandbox | The `steam://` link fallback (`SteamURL`), which opens through the OpenURI portal. Check that Flatpak'd and native Steam both take it |
| Desktop notifications (`notification_linux.go`, D-Bus) | Blocked unless allowed | `--talk-name=org.freedesktop.Notifications`, which Flathub accepts, or the notification portal |
| Plays sounds (miniaudio) | Needs the sound server | `--socket=pulseaudio` (PipeWire serves it too) |
| Everything else | | `--share=network`, `--socket=wayland`, `--socket=fallback-x11`, `--device=dri` |

#### Telling whether the game runs, as Chatterino does

Chatterino's Flatpak has the same problem: its streamer mode needs to know whether OBS is running. Its solution, in `src/singletons/StreamerMode.cpp`:

- **Outside a Flatpak**, it runs `pgrep -xi` with the program names joined by `|`.
- **Inside one**, it runs the same command through the host, with `flatpak-spawn --host pgrep -xi …`.
- **The permission.** This needs `--talk-name=org.freedesktop.Flatpak`, which its Flathub manifest (`flathub/com.chatterino.chatterino`, on `org.kde.Platform` 6.11) has.
- **Reading the result.** It waits up to a second for `pgrep`. Exit code 0 means running. A timeout, or `pgrep` failing to start, counts as not running, with a warning.

Lockdeader copies that, inside a Flatpak only (built):

- **Detecting the sandbox.** `deploy.Running` checks for `/.flatpak-info`, which Flatpak puts in every sandbox.
- **The check.** It runs `flatpak-spawn --host pgrep -xi deadlock.exe`, with a one-second timeout.
- **Exit codes.** 0 means running and 1 means not. Anything else, or a timeout, is an error, which the callers already treat as not knowing: the game watcher keeps its last answer, and applying goes ahead.
- **Outside a Flatpak**, it keeps reading `/proc`, which needs no other program.
- **Check with the real game under Proton:** that its process name, which `pgrep -x` matches, is `deadlock.exe`. The `/proc` code also accepts the first word of the command line, which `pgrep` would need `-f` and a pattern for.
- **Flathub's reviewers.** `org.freedesktop.Flatpak` lets the app run anything on the host, so they'll ask why it's needed. The manifest should say: it's to tell whether the game is running, as Chatterino's does for OBS.

The data directory moves to `~/.var/app/<app-id>/data/Lockdeader`, since Flatpak sets `XDG_DATA_HOME`; `appdata` follows it already. It's usually on the same file system as Steam's library in the home folder, so hard links still work.

### Getting onto Flathub

- **Submission.** Submit the manifest as a pull request to `flathub/flathub`. Reviewers check:
  - the app ID and our right to it;
  - the metainfo;
  - that each permission is needed (write down why next to each in the manifest);
  - that the app's name and description don't suggest it's Valve's.
- **Once accepted.** The app lives in its own repository under `flathub/`, which Flathub's bot updates from our tags if we set up `x-checker-data`.
- **Before submitting:** build and run the manifest ourselves, and ship it from GitHub Releases as a `.flatpakref` or bundle. Nothing about the Flatpak has to wait for Flathub. **Built:** the `flatpak` job of `.github/workflows/build.yml` runs `packaging/flatpak/build.ps1`, which makes a single-file bundle and uploads it as an artifact; ccache keeps miqt's compiled C++ between runs.

## Order of work

1. **Shared groundwork:** the app ID, embedding the QML, the desktop file, icons, metainfo and third-party notices.
2. **The Windows workflow:** build, collect DLLs, smoke-test, release. Windows is where most players are.
3. **An OBS home project with a Tumbleweed package**, then whichever other distributions have a new enough Qt.
4. **Making the app work in the sandbox:** telling whether the game is running (as Chatterino does), and the filesystem permissions. Then a Flatpak built from our own manifest.
5. **Submitting** to Flathub and to openSUSE, as each is ready.

## Decisions

| Question | Decision |
|---|---|
| App ID | `io.github.treejadey.Lockdeader` |
| Where the QML lives | Embedded in the binary |
| Which Qt for Windows | Whichever is more convenient; MSYS2's to start with |
| Telling whether the game runs in the Flatpak | As Chatterino does: the host's `pgrep`, through `flatpak-spawn --host` |

## Still open

These will be worked out later.

- **Flatpak game folders:** which locations get static permissions, and how libraries on other drives are reached.
- **Oldest Qt supported:** which also decides which distributions OBS can build for.
