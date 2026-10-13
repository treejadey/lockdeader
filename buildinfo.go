package main

/*
#cgo pkg-config: Qt6Core
#include <QtCore/qconfig.h>

static const char *lockdeader_qt_build_version(void) {
	return QT_VERSION_STR;
}
*/
import "C"

import (
	"fmt"
	"html"
	"runtime/debug"

	qt "github.com/mappu/miqt/qt6"
)

const commitURL = "https://github.com/treejadey/lockdeader/commit/"

// buildInfo describes this build and what it runs on, for the settings page,
// as rich text with a link to the commit it was built from, e.g.
//
//	Lockdeader 0.1.0 (commit 1bc6e5aa7) built with Qt 6.11.1 (running on 6.11.2)
//	Running on KDE Flatpak runtime, kernel: 7.2.9-cachyos1.lto.fc44.x86_64
//
// It isn't translated, so it reads the same in any bug report it's pasted into.
func buildInfo() string {
	build := "Lockdeader " + html.EscapeString(version)

	// Go stamps the commit when it builds from a Git checkout. The Flatpak
	// builds from a copy without .git, so it has none.
	if commit, modified := vcsRevision(); commit != "" {
		short := commit[:min(9, len(commit))]
		link := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(commitURL+commit), html.EscapeString(short))

		if modified {
			build += fmt.Sprintf(" (commit %s, modified)", link)
		} else {
			build += fmt.Sprintf(" (commit %s)", link)
		}
	}

	builtWith := C.GoString(C.lockdeader_qt_build_version())
	runningOn := qt.QLibraryInfo_Version().ToString()

	build += " built with Qt " + html.EscapeString(builtWith)

	if runningOn != builtWith {
		build += " (running on " + html.EscapeString(runningOn) + ")"
	}

	system := fmt.Sprintf("Running on %s, kernel: %s",
		html.EscapeString(qt.QSysInfo_PrettyProductName()), html.EscapeString(qt.QSysInfo_KernelVersion()))

	return build + "<br>" + system
}

func vcsRevision() (commit string, modified bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	return commit, modified
}
