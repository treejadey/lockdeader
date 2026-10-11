import QtQuick
import QtQuick.Controls.Fusion

// A window of the app's, in its theme: the main window, or a dialog shown
// before it.
//
// Every QML file imports QtQuick.Controls.Fusion rather than plain
// QtQuick.Controls, so the stock controls look the same on every desktop.
// Otherwise e.g. KDE Plasma substitutes its own style, which draws with the
// system palette and ignores the theme below.
ApplicationWindow {
    color: Theme.base

    // Dark palette for the stock controls (buttons, scroll bars, spinners, tooltips).
    palette {
        window: Theme.base
        windowText: Theme.text
        base: Theme.surface
        alternateBase: Theme.overlay
        text: Theme.text
        button: Theme.overlay
        buttonText: Theme.text
        highlight: Theme.highlightMed
        highlightedText: Theme.text
        light: Theme.highlightHigh
        midlight: Theme.highlightMed
        mid: Theme.overlay
        dark: Theme.base
        shadow: "black"
        placeholderText: Theme.muted
        toolTipBase: Theme.overlay
        toolTipText: Theme.text
    }
}
