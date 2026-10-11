import QtQuick
import QtQuick.Controls.Fusion

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
