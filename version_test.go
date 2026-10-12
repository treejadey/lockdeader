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
