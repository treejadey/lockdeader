package main

import (
	"embed"
	"fmt"
	"io/fs"

	qt "github.com/mappu/miqt/qt6"
)

// The fonts in fonts/, bundled so the app looks the same on every platform.
// fonts/README.md has their sources and licences.
//
//go:embed fonts/*/*.ttf
var fontFiles embed.FS

const (
	// The family name of the bundled Inter files, which have a stylistic set
	// built in and are renamed so they never mix with an installed Inter (see
	// fonts/README.md).
	uiFontFamily        = "Inter SS04"
	monospaceFontFamily = "JetBrains Mono"
)

// loadFonts makes the bundled fonts available to the app, and makes Inter its
// default font, at the platform's UI font size. It must be called after the application is created and before
// any QML is loaded.
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

// codeStyle is a stylesheet for rich text that sets code in the bundled
// monospace font, rather than the system's, at the size of the rest of the
// text. Qt's rich text ignores sizes in % or em, hence points.
func codeStyle() string {
	size := qt.QGuiApplication_Font().PointSizeF()
	return fmt.Sprintf(`<style>code, pre { font-family: "%s"; font-size: %.1fpt; }</style>`, monospaceFontFamily, size)
}
