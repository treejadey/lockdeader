# Flatpak

`io.github.treejadey.Lockdeader.yml` builds this checkout as a Flatpak, to
try it out. It isn't on Flathub; see `docs/distribution.md` for what that
would take.

## Building and running

From the repository's root, with `flatpak-builder` installed:

```sh
flatpak-builder --user --install-deps-from=flathub --force-clean --install \
    --state-dir=.flatpak-builder flatpak-build packaging/flatpak/io.github.treejadey.Lockdeader.yml
flatpak run io.github.treejadey.Lockdeader
```

`build.ps1` (PowerShell) does the same but writes a single
`dist/Lockdeader-<version>-linux-x86_64.flatpak` instead of installing,
with ccache keeping miqt's compiled C++ for the next build. GitHub Actions
runs it (`.github/workflows/build.yml`) and uploads the file. Install one
with `flatpak install --user dist/Lockdeader-*.flatpak`.

- **The first build** installs the KDE SDK and the Go extension from Flathub,
  then compiles miqt, which takes a while.
- **Later builds** start over in the sandbox, so they recompile too.
- **The installed app** shows up in the application menu like any other.
- **Removing it:** `flatpak uninstall --user io.github.treejadey.Lockdeader`.

## What's different in the sandbox

- **Data** is in `~/.var/app/io.github.treejadey.Lockdeader/data/Lockdeader`,
  not `~/.local/share/Lockdeader`.
- **Telling whether Deadlock is running** asks the host's `pgrep` through
  `flatpak-spawn --host`, which is what `--talk-name=org.freedesktop.Flatpak`
  is for.
- **Steam's folders** are reachable: `~/.local/share/Steam`, `~/.steam`, and
  Steam's own Flatpak. A Steam library on another drive isn't yet.
- **Go modules:** the build downloads them, which Flathub wouldn't allow.
