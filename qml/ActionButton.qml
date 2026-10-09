import QtQuick
import QtQuick.Controls.Fusion

// A regular push button: a centred label with an optional icon after it, or
// just an icon when there's no text.
// Following Carbon, the icon goes on the right and shows the action the
// button performs. End the label with "…" when the button opens a dialog to
// finish the action, rather than doing it straight away.
//
// Motion matches SidebarItem: Carbon's productive button micro-interaction,
// fast-01 (70 ms) with standard easing. Hovering lightens the background;
// pressing darkens it and shrinks the button slightly, as if pushed in.
AbstractButton {
    id: root

    implicitWidth: implicitContentWidth + leftPadding + rightPadding
    implicitHeight: 34
    leftPadding: 12
    rightPadding: 12
    hoverEnabled: true
    // Reachable with Tab, but clicking doesn't take focus from where it was.
    focusPolicy: Qt.TabFocus

    icon.width: 18
    icon.height: 18

    scale: down ? 0.97 : 1
    opacity: enabled ? 1 : 0.4

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
        color: root.down ? Theme.highlightLow
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

    contentItem: Item {
        implicitWidth: row.implicitWidth
        implicitHeight: row.implicitHeight

        Row {
            id: row

            anchors.centerIn: parent
            spacing: 8

            Label {
                anchors.verticalCenter: parent.verticalCenter
                visible: text !== ""
                text: root.text
                color: Theme.text
                font.weight: Font.DemiBold
            }

            Image {
                anchors.verticalCenter: parent.verticalCenter
                width: root.icon.width
                height: root.icon.height
                visible: root.icon.source.toString() !== ""
                source: root.icon.source
                sourceSize: Qt.size(width, height)
            }
        }
    }
}
