import QtQuick
import QtQuick.Controls.Fusion

// Where it goes is up to its parent.
Rectangle {
    id: root

    property alias text: label.text
    property bool shown: false
    // In milliseconds.
    property int duration: 2000
    property real maximumWidth: 420

    function show(message) {
        text = message
        shown = true
        timer.restart()
    }

    width: Math.min(label.implicitWidth + 24, maximumWidth)
    height: label.implicitHeight + 12
    radius: 6
    color: Theme.overlay
    border.color: Theme.highlightMed
    opacity: 0
    visible: opacity > 0

    transform: Translate {
        id: shift

        y: -8
    }

    Label {
        id: label

        anchors.centerIn: parent
        width: Math.min(implicitWidth, root.maximumWidth - 24)
        color: Theme.text
        font.weight: Font.DemiBold
        elide: Text.ElideRight
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

        interval: root.duration
        onTriggered: root.shown = false
    }
}
