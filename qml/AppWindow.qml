import QtQuick
import QtQuick.Controls.Fusion

// Every QML file imports QtQuick.Controls.Fusion rather than plain
// QtQuick.Controls: otherwise e.g. KDE Plasma substitutes its own style, which
// draws with the system palette and ignores the one below.
ApplicationWindow {
    color: Theme.base

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
