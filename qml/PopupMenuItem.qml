import QtQuick
import QtQuick.Controls.Fusion

MenuItem {
    id: root

    // Hides the item, with no gap left where it was: a hidden item would
    // otherwise still take up its height in the menu. Use this rather than
    // visible, which is false whenever the menu is closed, so a height bound
    // to it changes as the menu opens, resizing the menu: a binding loop.
    property bool shown: true

    visible: shown
    height: shown ? implicitHeight : 0

    leftPadding: 12
    rightPadding: 16
    topPadding: 8
    bottomPadding: 8
    spacing: 10

    icon.width: 18
    icon.height: 18
    // Qt colours a one-colour icon to match. The SVGs' own fill is only right
    // for the dark style.
    icon.color: Theme.text

    background: Rectangle {
        radius: 8
        color: root.highlighted ? Theme.highlightMed : "transparent"

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }
}
