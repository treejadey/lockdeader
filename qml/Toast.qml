import QtQuick
import QtQuick.Controls.Fusion

// Where it goes is up to its parent.
Rectangle {
    id: root

    property alias text: label.text
    // The action button's label, e.g. "Undo"; "" for none.
    property string action: ""
    property bool shown: false
    // In milliseconds. A toast with an action stays up longer, to give time to
    // reach it.
    property int duration: 2000
    property int actionDuration: 6000
    property real maximumWidth: 420

    readonly property bool hasAction: action !== ""
    readonly property real rightPadding: hasAction ? 4 : 12

    signal actionTriggered()
    // The message went without its action being used: it timed out, was
    // closed, or another replaced it.
    signal dismissed()

    function show(message, action) {
        if (shown)
            dismissed()

        text = message
        root.action = action ?? ""
        shown = true
        timer.restart()
    }

    function trigger() {
        if (!shown || !hasAction)
            return

        shown = false
        timer.stop()

        actionTriggered()
    }

    function close() {
        if (!shown)
            return

        shown = false
        timer.stop()

        dismissed()
    }

    width: Math.min(content.implicitWidth + 12 + rightPadding, maximumWidth)
    height: hasAction ? 36 : label.implicitHeight + 12
    radius: 6
    color: Theme.overlay
    border.color: Theme.highlightMed
    opacity: 0
    visible: opacity > 0

    transform: Translate {
        id: shift

        y: -8
    }

    Row {
        id: content

        anchors {
            left: parent.left
            leftMargin: 12
            verticalCenter: parent.verticalCenter
        }
        spacing: 2

        Label {
            id: label

            anchors.verticalCenter: parent.verticalCenter
            width: Math.min(implicitWidth, root.maximumWidth - 12 - root.rightPadding
                - (root.hasAction ? actionButton.width + closeButton.width + 2 * content.spacing : 0))
            // Clear of the action button by the 12 px a label keeps from its control.
            rightPadding: root.hasAction ? 12 - content.spacing : 0
            color: Theme.text
            font.weight: Font.DemiBold
            elide: Text.ElideRight
        }

        AbstractButton {
            id: actionButton

            anchors.verticalCenter: parent.verticalCenter
            visible: root.hasAction
            height: 28
            leftPadding: 8
            rightPadding: 8
            hoverEnabled: true
            focusPolicy: Qt.TabFocus
            text: root.action
            onClicked: root.trigger()

            HoverHandler {
                cursorShape: Qt.PointingHandCursor
            }

            background: ToastButtonBackground {
                button: actionButton
            }

            contentItem: Label {
                text: actionButton.text
                color: Theme.foam
                font.weight: Font.DemiBold
                verticalAlignment: Text.AlignVCenter
            }
        }

        AbstractButton {
            id: closeButton

            anchors.verticalCenter: parent.verticalCenter
            visible: root.hasAction
            width: 28
            height: 28
            padding: 7
            hoverEnabled: true
            focusPolicy: Qt.TabFocus
            onClicked: root.close()

            HoverHandler {
                cursorShape: Qt.PointingHandCursor
            }

            ToolTip.visible: hovered
            ToolTip.delay: 500
            ToolTip.text: qsTr("Close")

            background: ToastButtonBackground {
                button: closeButton
            }

            contentItem: TintedIcon {
                source: "icons/dismiss_24_regular.svg"
                opacity: closeButton.hovered ? 1 : 0.8
            }
        }
    }

    component ToastButtonBackground: Rectangle {
        required property AbstractButton button

        radius: 4
        color: button.down ? Theme.highlightHigh
             : button.hovered ? Theme.highlightMed
             : "transparent"
        border.width: button.visualFocus ? 1 : 0
        border.color: Theme.iris

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    states: State {
        name: "shown"
        when: root.shown

        PropertyChanges {
            root.opacity: 1
            shift.y: 0
        }
    }

    // Carbon counts notifications as expressive moments, so it pops in with
    // an expressive entrance, then leaves with a quick productive exit.
    transitions: [
        Transition {
            to: "shown"

            NumberAnimation {
                targets: [root, shift]
                properties: "opacity,y"
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }
        },

        Transition {
            from: "shown"

            NumberAnimation {
                targets: [root, shift]
                properties: "opacity,y"
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }
        }
    ]

    Timer {
        id: timer

        interval: root.hasAction ? root.actionDuration : root.duration
        onTriggered: root.close()
    }
}
