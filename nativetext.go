package main

/*
#cgo pkg-config: Qt6Quick
// GCC 16 warns about this in Qt's own headers; nothing to fix on our side.
#cgo CXXFLAGS: -Wno-sfinae-incomplete
#include "nativetext.h"
*/
import "C"

// useNativeTextRendering makes Qt Quick draw text with the platform's font
// rasterizer (FreeType on Linux), like other desktop apps, instead of Qt's own
// renderer. Native text is snapped to whole pixels, so it stays sharp on
// fractionally scaled displays (e.g. 125%); Qt's renderer goes soft there, by
// an amount that depends on where the text lands, so it changes as the window
// is resized.
//
// It must be called before any QML is loaded. It's written in C++
// (nativetext.cpp) because miqt has no bindings for Qt Quick.
func useNativeTextRendering() {
	C.lockdeader_use_native_text_rendering()
}
