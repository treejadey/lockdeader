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
