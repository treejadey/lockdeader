package main

import qt "github.com/mappu/miqt/qt6"

// tr is Go's qsTr, for text that Go hands to the UI. lupdate can't read Go, so
// these strings will need extracting some other way once there are
// translations.
func tr(text string) string {
	return qt.QCoreApplication_Translate("Lockdeader", text)
}
