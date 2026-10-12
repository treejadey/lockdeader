package main

/*
#cgo pkg-config: Qt6Core
// GCC 16 warns about this in Qt's own headers; nothing to fix on our side.
#cgo CXXFLAGS: -Wno-sfinae-incomplete
#include <stdlib.h>
#include "translator.h"
*/
import "C"

import (
	"embed"
	"io/fs"
	"log/slog"
	"sync/atomic"

	qt "github.com/mappu/miqt/qt6"

	"github.com/treejadey/lockdeader/i18n"
)

// Bundled, so they're found wherever the app is.
//
//go:embed po
var translationFiles embed.FS

// installTranslations translates the app into the system's language, the
// first of the user's languages there's a translation for, or leaves it in
// English. It must be called before any QML is loaded.
func installTranslations() {
	// On Linux, this follows LANGUAGE (e.g. "de:fr"), then LC_ALL,
	// LC_MESSAGES and LANG; on Windows, the display language.
	languages := qt.QLocale_System().UiLanguages()

	t, err := i18n.Load(translationFS(), languages)
	if err != nil {
		slog.Error("failed to load the translation, so the app is in English", "err", err)
	}

	slog.Info("translating", "languages", languages, "into", t)

	// Even for English, which needs it for plurals.
	installTranslator(t)

	// Qt's own text, e.g. text fields' menus, in the same language. Qt asks
	// the translator installed last first, and this one passes on text it
	// doesn't have.
	if t.Language != "" {
		qtTranslator := qt.NewQTranslator()

		if qtTranslator.Load8(qt.NewQLocale2(t.Language), "qt", "_", qt.QLibraryInfo_Path(qt.QLibraryInfo__TranslationsPath)) {
			qt.QCoreApplication_InstallTranslator(qtTranslator)
		} else {
			slog.Info("Qt has no translation of its own text into the language", "language", t.Language)
		}
	}
}

func translationFS() fs.FS {
	files, err := fs.Sub(translationFiles, "po")
	if err != nil {
		panic(err)
	}

	return files
}

// The translator that Qt's translate calls go to. Qt may call from any
// thread.
var translator atomic.Pointer[i18n.Translator]

// installTranslator hands Qt's translate calls to t (see translator.cpp).
// Its translator always answers, so Qt never asks one installed before it.
func installTranslator(t *i18n.Translator) {
	translator.Store(t)
	C.lockdeader_install_translator()
}

//export lockdeaderTranslate
func lockdeaderTranslate(source, disambiguation *C.char, n C.int) *C.char {
	return C.CString(translator.Load().Translate(C.GoString(source), C.GoString(disambiguation), int(n)))
}
