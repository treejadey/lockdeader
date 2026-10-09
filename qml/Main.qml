import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// Every QML file imports QtQuick.Controls.Fusion rather than plain
// QtQuick.Controls, so the stock controls look the same on every desktop.
// Otherwise e.g. KDE Plasma substitutes its own style, which draws with the
// system palette and ignores the theme below.
ApplicationWindow {
    id: window

    width: 1100
    height: 720
    visible: true
    title: "lockdeader"
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

    RowLayout {
        anchors.fill: parent
        spacing: 0

        Sidebar {
            id: sidebar
            Layout.preferredWidth: 208
            Layout.fillHeight: true
        }

        StackLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: sidebar.currentIndex

            ModsPage {}

            PlaceholderPage {
                title: "Installed mods"
                text: "Mods you install will show up here."
            }

            PlaceholderPage {
                title: "Settings"
                text: "Nothing to configure yet."
            }
        }
    }
}
