package main

import (
	"embed"
	"fmt"
	"io/fs"

	qt "github.com/mappu/miqt/qt6"
)

// Bundled so the app looks the same on every platform. fonts/README.md has
// their sources and licences.
//
//go:embed fonts/*/*.ttf
var fontFiles embed.FS

const (
	// The bundled Inter files have a stylistic set built in, and are renamed
	// so they never mix with an installed Inter (see fonts/README.md).
	uiFontFamily        = "Inter SS04"
	monospaceFontFamily = "JetBrains Mono"
)

// loadFonts must be called after the application is created and before any
// QML is loaded.
func loadFonts() error {
	names, err := fs.Glob(fontFiles, "fonts/*/*.ttf")
	if err != nil {
		return err
	}

	for _, name := range names {
		data, err := fontFiles.ReadFile(name)
		if err != nil {
			return err
		}

		if qt.QFontDatabase_AddApplicationFontFromData(data) < 0 {
			return fmt.Errorf("loading font %s: Qt rejected it", name)
		}
	}

	font := qt.NewQFont2(uiFontFamily)
	font.SetPointSizeF(qt.QGuiApplication_Font().PointSizeF())
	qt.QGuiApplication_SetFont(font)

	return nil
}

// codeStyle sets code in rich text in the bundled monospace font, rather than
// the system's. Qt's rich text ignores sizes in % or em, hence points.
func codeStyle() string {
	size := qt.QGuiApplication_Font().PointSizeF()

	return fmt.Sprintf(`<style>code, pre { font-family: "%s"; font-size: %.1fpt; }</style>`, monospaceFontFamily, size)
}
