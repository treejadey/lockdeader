package main

import (
	"regexp"
	"testing"
)

// The version goes into the user agent and packages' metadata, which expect
// MAJOR.MINOR.PATCH.
func TestVersion(t *testing.T) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(version) {
		t.Errorf("VERSION.txt holds %q, want MAJOR.MINOR.PATCH", version)
	}
}

func TestDisplayVersion(t *testing.T) {
	defer func(saved string) { buildType = saved }(buildType)

	buildType = ""
	if got := displayVersion(); got != version {
		t.Errorf("displayVersion of a local build = %q, want %q", got, version)
	}

	buildType = "nightly"
	if got := displayVersion(); got != version+" Nightly" {
		t.Errorf("displayVersion of a nightly = %q", got)
	}
}

// Builds Go can't stamp a commit on, like the Flatpak, and test binaries,
// get it from buildCommit.
func TestVCSRevisionFallback(t *testing.T) {
	defer func(saved string) { buildCommit = saved }(buildCommit)

	buildCommit = "0123456789abcdef"

	if commit, _ := vcsRevision(); commit != buildCommit {
		t.Errorf("vcsRevision = %q, want %q", commit, buildCommit)
	}
}
