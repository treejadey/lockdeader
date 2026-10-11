import QtQuick
import QtQuick.Controls.Fusion

// Not named Dialog: Qt Quick Controls has a type of that name already.
//
// Items declared in an AppDialog are its content, stacked top to bottom;
// put its buttons in buttons, from left to right.
//
// Motion is the install dialog's: opening a modal is an expressive moment, so
// the card rises from low down towards the middle of the window as it fades
// in, with Carbon's expressive entrance, over slow-01, as it moves a long
// way. The backdrop dims over Carbon's background dimming duration. Closing
// is a quick productive exit that drops it back slightly.
Popup {
    id: root

    property string title
    // Dimmer, e.g. the mod's name in "Yamato configuration".
    property string titlePrefix
    property int cardWidth: 560

    default property alias content: body.data
    property alias buttons: buttonRow.data

    parent: Overlay.overlay
    x: 0
    y: 0
    width: parent ? parent.width : 0
    height: parent ? parent.height : 0
    padding: 0

    modal: true
    dim: false
    focus: true
    closePolicy: Popup.CloseOnEscape
    background: null

    enter: Transition {
        ParallelAnimation {
            NumberAnimation {
                target: backdrop
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.slow02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }

            NumberAnimation {
                target: card
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.slow01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }

            // From most of the way down to the bottom of the window.
            NumberAnimation {
                target: cardShift
                property: "y"
                from: (root.height - card.y) * 0.6
                to: 0
                duration: Motion.slow01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }
        }
    }

    exit: Transition {
        ParallelAnimation {
            NumberAnimation {
                targets: [backdrop, card]
                property: "opacity"
                to: 0
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }

            NumberAnimation {
                target: cardShift
                property: "y"
                to: 24
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }
        }
    }

    Rectangle {
        id: backdrop

        anchors.fill: parent
        color: Theme.backdrop

        // A MouseArea rather than a TapHandler: pointer handlers see every
        // press under them, even ones the dialog above has already taken.
        MouseArea {
            anchors.fill: parent
            onClicked: root.close()
        }
    }

    Rectangle {
        id: card

        anchors.centerIn: parent
        width: Math.min(root.cardWidth, parent.width - 48)
        height: body.y + body.implicitHeight
              + (buttonRow.children.length > 0 ? 36 + buttonRow.height + 14 : 28)
        radius: 22
        color: Theme.overlay
        // Keeps text from wobbling on fractionally scaled displays. The dialog
        // is a popup, so the nudge on the main window's content doesn't reach it.
        transform: [
            Translate {
                id: cardShift
            },
            PixelNudge {}
        ]

        // Clicks on the dialog stay on it, rather than closing it.
        MouseArea {
            anchors.fill: parent
        }

        // The title, with its dimmer start cut short first when it's long.
        Row {
            id: titleRow

            anchors {
                left: parent.left
                leftMargin: 24
                verticalCenter: closeButton.verticalCenter
            }
            spacing: prefixLabel.visible ? 5 : 0

            Label {
                id: prefixLabel

                width: Math.min(implicitWidth, Math.max(0,
                    card.width - titleRow.anchors.leftMargin - titleLabel.implicitWidth
                    - titleRow.spacing - closeButton.width - 2 * closeButton.anchors.rightMargin))
                visible: root.titlePrefix !== ""
                text: root.titlePrefix
                color: Theme.subtle
                font.pixelSize: 15
                font.weight: Font.DemiBold
                elide: Text.ElideRight
            }

            Label {
                id: titleLabel

                text: root.title
                color: Theme.text
                font.pixelSize: 15
                font.weight: Font.DemiBold
            }
        }

        Column {
            id: body

            anchors {
                top: parent.top
                topMargin: 62
                left: parent.left
                right: parent.right
                leftMargin: 30
                rightMargin: 30
            }
            spacing: 14
        }

        Row {
            id: buttonRow

            anchors {
                right: parent.right
                bottom: parent.bottom
                margins: 14
            }
            spacing: 8
        }

        AbstractButton {
            id: closeButton

            anchors {
                top: parent.top
                right: parent.right
                topMargin: 14
                rightMargin: 16
            }
            width: 32
            height: 32
            padding: 7
            hoverEnabled: true
            focusPolicy: Qt.TabFocus
            onClicked: root.close()

            HoverHandler {
                cursorShape: Qt.PointingHandCursor
            }

            ToolTip.visible: hovered
            ToolTip.delay: 500
            ToolTip.text: qsTr("Close (Esc)")

            background: Rectangle {
                radius: 6
                color: closeButton.hovered ? Theme.highlightMed : "transparent"
                border.width: closeButton.visualFocus ? 1 : 0
                border.color: Theme.iris

                Behavior on color {
                    ColorAnimation {
                        duration: Motion.fast01
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            contentItem: TintedIcon {
                source: "icons/dismiss_24_regular.svg"
                opacity: closeButton.hovered ? 1 : 0.8
            }
        }
    }
}
