import QtQuick
import QtQuick.Controls.Fusion

// A navigation button in the sidebar: an icon and a label.
//
// Hover and press feedback is Carbon's productive motion for button
// micro-interactions: fast-01 (70 ms) with standard easing. Hovering lightens
// the background and brightens the icon and label; pressing darkens the
// background and shrinks the button slightly, as if pushed in.
AbstractButton {
    id: root

    // The page this button leads to is the one being shown.
    property bool active: false

    implicitHeight: 34
    leftPadding: 8
    rightPadding: 8
    hoverEnabled: true
    // Reachable with Tab, but clicking doesn't take focus from where it was.
    focusPolicy: Qt.TabFocus

    scale: down ? 0.97 : 1

    Behavior on scale {
        NumberAnimation {
            duration: Motion.fast01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }

    background: Rectangle {
        radius: 6
        color: root.active ? Theme.highlightMed
             : root.down ? Theme.highlightLow
             : root.hovered ? Qt.lighter(Theme.overlay, 1.25)
             : Theme.overlay
        // Only shown when focused with the keyboard.
        border.width: root.visualFocus ? 1 : 0
        border.color: Theme.iris

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    contentItem: Row {
        spacing: 8

        Image {
            anchors.verticalCenter: parent.verticalCenter
            width: 20
            height: 20
            source: root.icon.source
            sourceSize: Qt.size(width, height)
            opacity: root.active || root.hovered ? 1 : 0.6

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        Label {
            anchors.verticalCenter: parent.verticalCenter
            text: root.text
            color: root.active || root.hovered ? Theme.text : Theme.subtle
            font.weight: root.active ? Font.DemiBold : Font.Normal

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
    }
}
