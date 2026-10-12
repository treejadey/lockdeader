pragma Singleton

import QtQuick

// App-wide colours: Rosé Pine while the system prefers a dark style, and
// Rosé Pine Dawn otherwise. As elementary OS asks of apps that work in
// either, it follows the system's preference, and changes along with it.
// Where the system states no preference (Windows' high-contrast themes, a
// Linux desktop without a settings portal), its palette decides: a window
// darker than its text means dark.
QtObject {
    id: theme

    readonly property SystemPalette system: SystemPalette {}
    readonly property bool dark: Application.styleHints.colorScheme === Qt.ColorScheme.Dark
                                 || (Application.styleHints.colorScheme === Qt.ColorScheme.Unknown
                                     && theme.system.window.hslLightness < theme.system.windowText.hslLightness)

    readonly property color base: theme.dark ? "#191724" : "#faf4ed"
    readonly property color surface: theme.dark ? "#1f1d2e" : "#fffaf3"
    readonly property color overlay: theme.dark ? "#26233a" : "#f2e9e1"
    readonly property color highlightLow: theme.dark ? "#21202e" : "#f4ede8"
    readonly property color highlightMed: theme.dark ? "#403d52" : "#dfdad9"
    readonly property color highlightHigh: theme.dark ? "#524f67" : "#cecacd"

    readonly property color text: theme.dark ? "#e0def4" : "#575279"
    readonly property color subtle: theme.dark ? "#908caa" : "#797593"
    readonly property color muted: theme.dark ? "#6e6a86" : "#9893a5"

    readonly property color gold: theme.dark ? "#f6c177" : "#ea9d34"
    readonly property color rose: theme.dark ? "#ebbcba" : "#d7827e"
    readonly property color love: theme.dark ? "#eb6f92" : "#b4637a"
    readonly property color pine: theme.dark ? "#31748f" : "#286983"
    readonly property color foam: theme.dark ? "#9ccfd8" : "#56949f"
    readonly property color iris: theme.dark ? "#c4a7e7" : "#907aa9"

    // Fills the main action, e.g. Install, with textOnAccent text on it. Foam
    // in both styles; its text is Rosé Pine's darkest, as light text would be
    // too faint on Dawn's foam.
    readonly property color accent: theme.foam
    readonly property color textOnAccent: "#191724"

    // Behind text being typed into: set into the row in the dark style,
    // standing out from it in the light one.
    readonly property color field: theme.dark ? theme.highlightLow : theme.surface

    // Dims the window behind a dialog or the image viewer. Dark in both
    // styles, as in the dark one, so what's on top stands out; text drawn
    // straight on it uses textOnBackdrop.
    readonly property color backdrop: Qt.rgba(0x19 / 255, 0x17 / 255, 0x24 / 255, 0.92)
    readonly property color textOnBackdrop: "#e0def4"

    // Text and icons on top of pictures, e.g. a mod's name on its card, are
    // light whatever the style, over a dark shade.
    readonly property color textOnPicture: "white"

    // For disabled controls (see DisabledFill): brighter at the top, darker at
    // the bottom. Small marks, such as a checkbox, use the Mark colours, which
    // stand out from the fill so they still show up.
    readonly property color disabledTop: theme.dark ? "#423f53" : "#f6f1ec"
    readonly property color disabledBottom: theme.dark ? "#302e3e" : "#e6e0db"
    readonly property color disabledMarkTop: theme.dark ? "#7e7c8a" : "#c4bfca"
    readonly property color disabledMarkBottom: theme.dark ? "#757380" : "#b8b3bf"
    readonly property color disabledText: theme.muted

    // A colour a little further from the background, for hovering:
    // lighter in the dark style, darker in the light one, which needs less to
    // show.
    function lift(color, amount) {
        return theme.dark ? Qt.lighter(color, 1 + amount) : Qt.darker(color, 1 + amount / 3)
    }
}
