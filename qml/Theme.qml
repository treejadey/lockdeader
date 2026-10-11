pragma Singleton

import QtQuick

// App-wide colours: Rosé Pine while the system prefers a dark style, and
// Rosé Pine Dawn otherwise. As elementary OS asks of apps that work in
// either, it follows the system's preference, and changes along with it.
// With no preference, it's light, as elementary's apps are by default.
QtObject {
    id: theme

    readonly property bool dark: Application.styleHints.colorScheme === Qt.ColorScheme.Dark

    readonly property color base: dark ? "#191724" : "#faf4ed"
    readonly property color surface: dark ? "#1f1d2e" : "#fffaf3"
    readonly property color overlay: dark ? "#26233a" : "#f2e9e1"
    readonly property color highlightLow: dark ? "#21202e" : "#f4ede8"
    readonly property color highlightMed: dark ? "#403d52" : "#dfdad9"
    readonly property color highlightHigh: dark ? "#524f67" : "#cecacd"

    readonly property color text: dark ? "#e0def4" : "#575279"
    readonly property color subtle: dark ? "#908caa" : "#797593"
    readonly property color muted: dark ? "#6e6a86" : "#9893a5"

    readonly property color gold: dark ? "#f6c177" : "#ea9d34"
    readonly property color rose: dark ? "#ebbcba" : "#d7827e"
    readonly property color love: dark ? "#eb6f92" : "#b4637a"
    readonly property color pine: dark ? "#31748f" : "#286983"
    readonly property color foam: dark ? "#9ccfd8" : "#56949f"
    readonly property color iris: dark ? "#c4a7e7" : "#907aa9"

    // Fills the main action, e.g. Install, with onAccent text on it. Dawn's
    // foam is too light for light text, so the light style uses pine.
    readonly property color accent: dark ? foam : pine
    readonly property color onAccent: base

    // Behind text being typed into: set into the row in the dark style,
    // standing out from it in the light one.
    readonly property color field: dark ? highlightLow : surface

    // Text and icons on top of pictures, e.g. a mod's name on its card, are
    // light whatever the style, over a dark shade.
    readonly property color onPicture: "white"

    // For disabled controls (see DisabledFill): brighter at the top, darker at
    // the bottom. Small marks, such as a checkbox, use the Mark colours, which
    // stand out from the fill so they still show up.
    readonly property color disabledTop: dark ? "#423f53" : "#f6f1ec"
    readonly property color disabledBottom: dark ? "#302e3e" : "#e6e0db"
    readonly property color disabledMarkTop: dark ? "#7e7c8a" : "#c4bfca"
    readonly property color disabledMarkBottom: dark ? "#757380" : "#b8b3bf"
    readonly property color disabledText: muted

    // A colour a little further from the background, for hovering:
    // lighter in the dark style, darker in the light one, which needs less to
    // show.
    function lift(color, amount) {
        return dark ? Qt.lighter(color, 1 + amount) : Qt.darker(color, 1 + amount / 3)
    }
}
