package main

/*
#cgo pkg-config: Qt6Quick
// GCC 16 warns about this in Qt's own headers; nothing to fix on our side.
#cgo CXXFLAGS: -Wno-sfinae-incomplete
#include "nativetext.h"
*/
import "C"

// Native text, drawn by the platform's font rasterizer (FreeType on Linux), is
// snapped to whole pixels, so it stays sharp on fractionally scaled displays
// (e.g. 125%). Qt's own renderer goes soft there, by an amount that depends on
// where the text lands, so it changes as the window is resized.
//
// It must be called before any QML is loaded. It's written in C++
// (nativetext.cpp) because miqt has no bindings for Qt Quick.
func useNativeTextRendering() {
	C.lockdeader_use_native_text_rendering()
}
