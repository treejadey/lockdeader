import QtQuick
import QtQuick.Controls.Fusion

// Following Carbon, the icon goes after the label and shows the action the
// button performs. End the label with "…" when the button opens a dialog to
// finish the action, rather than doing it straight away.
AbstractButton {
    id: root

    // Overlay suits the panels; on something already overlay, e.g. a settings
    // row, use highlightMed.
    property color baseColor: Theme.overlay
    // E.g. Theme.base on a light baseColor such as foam.
    property color textColor: Theme.text
    // Shades the button from nearly white at the top down to baseColor, for
    // an action that should stand out from everything around it.
    property bool sheen: false

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
        id: background

        radius: 6
        // Clear while disabled, so its antialiased edge doesn't show around
        // the disabled look's.
        color: !root.enabled ? Qt.rgba(root.baseColor.r, root.baseColor.g, root.baseColor.b, 0)
             : root.down ? Qt.darker(root.baseColor, 1.15)
             : root.hovered ? Qt.lighter(root.baseColor, 1.25)
             : root.baseColor
        // Follows color, so it lightens and darkens with hovering and pressing.
        gradient: root.sheen && root.enabled ? sheenGradient : null
        border.width: root.visualFocus ? 1 : 0
        border.color: Theme.iris

        Gradient {
            id: sheenGradient

            GradientStop {
                position: 0
                color: Qt.tint(background.color, Qt.rgba(1, 1, 1, 0.45))
            }

            GradientStop {
                position: 1
                color: background.color
            }
        }

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        DisabledFill {
            anchors.fill: parent
            radius: parent.radius
            shown: !root.enabled
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
                color: root.enabled ? root.textColor : Theme.disabledText
                font.weight: Font.DemiBold

                Behavior on color {
                    ColorAnimation {
                        duration: Motion.fast01
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            Image {
                anchors.verticalCenter: parent.verticalCenter
                width: root.icon.width
                height: root.icon.height
                visible: root.icon.source.toString() !== ""
                source: root.icon.source
                sourceSize: Qt.size(width, height)
                opacity: root.enabled ? 1 : 0.4
            }
        }
    }
}
