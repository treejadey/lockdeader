package main

import (
	_ "embed"
	"strings"
)

// The app's version, MAJOR.MINOR.PATCH. VERSION.txt is the one place it's
// set; packaging reads it from there too.
//
//go:embed VERSION.txt
var versionFile string

var version = strings.TrimSpace(versionFile)

// Set by the build scripts in packaging/ with -ldflags -X, and empty in a
// local build. buildType is "nightly" for a build that isn't from a v* tag,
// as Chatterino marks its nightlies. buildCommit is the commit, for builds Go
// can't stamp it on: the Flatpak builds from a copy without .git.
var (
	buildType   string
	buildCommit string
)

// displayVersion is the version as the app shows it, which says when it's a
// nightly build.
func displayVersion() string {
	if buildType == "nightly" {
		return version + " Nightly"
	}

	return version
}
