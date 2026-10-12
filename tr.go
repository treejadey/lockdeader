package main

import qt "github.com/mappu/miqt/qt6"

// tr is Go's qsTr, for text that Go hands to the UI. The translations tool
// (cmd/translations) finds its calls, so the text must be a string literal.
func tr(text string) string {
	return qt.QCoreApplication_Translate("Lockdeader", text)
}

// trN is tr for text that depends on a number, with %n where Qt puts it:
// trN("%n mod", "%n mods", count). The translation has as many forms as its
// language needs.
func trN(one, many string, n int) string {
	return qt.QCoreApplication_Translate3("Lockdeader", one, many, n)
}
