package main

/*
#cgo pkg-config: Qt6Qml Qt6Network
// GCC 16 warns about this in Qt's own headers; nothing to fix on our side.
#cgo CXXFLAGS: -Wno-sfinae-incomplete
#include <stdlib.h>
#include "imagecache.h"
*/
import "C"

import (
	"strings"
	"unsafe"

	"github.com/mappu/miqt/qt6/qml"
)

// installImageCache gives the QML engine an on-disk cache of at most maxBytes
// in dir, for the images it loads. Images from immutableHosts are served from
// the cache without asking the server whether they've changed.
//
// It's written in C++ (imagecache.cpp) because miqt deliberately doesn't bind
// QQmlNetworkAccessManagerFactory::create. It must be called before any QML
// is loaded.
func installImageCache(engine *qml.QQmlEngine, dir string, maxBytes int64, immutableHosts []string) {
	cDir := C.CString(dir)
	defer C.free(unsafe.Pointer(cDir))
	cHosts := C.CString(strings.Join(immutableHosts, ","))
	defer C.free(unsafe.Pointer(cHosts))

	C.lockdeader_install_image_cache(engine.UnsafePointer(), cDir, C.longlong(maxBytes), cHosts)
}
