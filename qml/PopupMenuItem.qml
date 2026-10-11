import QtQuick
import QtQuick.Controls.Fusion

// An entry in a PopupMenu: an icon and a label, with room around them.
// The hover highlight is Carbon's productive button micro-interaction:
// fast-01 (70 ms) with standard easing.
MenuItem {
    id: root

    leftPadding: 12
    rightPadding: 16
    topPadding: 8
    bottomPadding: 8
    spacing: 10
    icon.width: 18
    icon.height: 18

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
