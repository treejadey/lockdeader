import QtQuick
import QtQuick.Effects

// Declare it just before the text so it sits underneath, as a sibling of it:
//
//     TextShadow { source: title }
//     Label { id: title; ... }
//
// The text itself is left alone. Using MultiEffect's own shadow instead would
// render the text through an effect layer, which blurs it on fractionally
// scaled displays.
MultiEffect {
    x: source ? source.x : 0
    y: source ? source.y + 1 : 0
    width: source ? source.width : 0
    height: source ? source.height : 0

    blurEnabled: true
    blur: 1.0
    blurMax: 4
    brightness: -1.0
    opacity: 0.6
}
