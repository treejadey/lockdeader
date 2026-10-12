# Windows zip

`build.ps1` builds `Lockdeader-<version>-windows-x64.zip`: a `Lockdeader`
folder with the app and everything it needs, so it runs where it's unzipped.
`smoke-test.ps1` checks that it does, on a machine with nothing installed.
GitHub Actions runs both (`.github/workflows/build.yml`).

## Building by hand

1. Install [MSYS2](https://www.msys2.org/), and in its UCRT64 shell the
   packages the workflow lists: `gcc`, `pkgconf`, `qt6-base`,
   `qt6-declarative`, `qt6-svg`, `qt6-imageformats`, `qt6-translations`,
   `qt6-tools` (for `lconvert`, which `windeployqt` merges the translations
   with) and `ntldd`, each as `mingw-w64-ucrt-x86_64-<name>`.
2. Install [Go](https://go.dev/dl/).
3. From the repository's root, in PowerShell:

   ```powershell
   ./packaging/windows/build.ps1 -Msys2Root C:\msys64
   ./packaging/windows/smoke-test.ps1 -Zip (Get-ChildItem dist/*.zip).FullName
   ```

## What's in the zip, and how it's found

- **`Lockdeader.exe`**, built without a console window. The QML is inside
  it.
- **Qt's DLLs, plugins and QML modules**, placed by `windeployqt` as Qt
  looks for them: plugins in `platforms/`, `imageformats/` and so on next to
  the app, and QML modules such as `QtQuick/` next to it too.
- **The libraries Qt itself uses** from MSYS2 (ICU, zstd, HarfBuzz, the
  MinGW runtime, ...), which `windeployqt` doesn't copy. `ntldd` reads each
  file's imports, and anything it finds in MSYS2's `ucrt64\bin` is copied
  in, until nothing new turns up.
- **Qt's own translations** in `translations/`, where `translations.go`
  looks after Qt's install folder.
- **`LICENSE.txt`.** Licences for what's bundled are still to come (see
  `docs/distribution.md`).
