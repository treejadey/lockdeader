package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"embed"
	"errors"
	"unsafe"

	qt "github.com/mappu/miqt/qt6"

	"github.com/treejadey/lockdeader/rcc"
)

// The QML, its icons and pictures, bundled so the app finds them wherever
// it's started from or installed.
//
//go:embed qml
var qmlFiles embed.FS

var errQMLNotRegistered = errors.New("Qt didn't take the bundled QML")

// registerQML makes qml/ readable at qrc:/qml/. It must be called before any
// QML is loaded.
func registerQML() error {
	data, err := rcc.Build(qmlFiles)
	if err != nil {
		return err
	}

	// Qt reads the data for as long as the app runs, so it's copied where
	// Go's garbage collector won't free it, and never freed.
	rccData := C.CBytes(data)

	if !qt.QResource_RegisterResourceWithRccData((*byte)(rccData)) {
		C.free(unsafe.Pointer(rccData))

		return errQMLNotRegistered
	}

	return nil
}

// qmlURL is where a file in qml/ is, e.g. qmlURL("Main.qml").
func qmlURL(name string) *qt.QUrl {
	return qt.NewQUrl3("qrc:/qml/" + name)
}
