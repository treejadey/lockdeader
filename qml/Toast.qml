import QtQuick
import QtQuick.Controls.Fusion

// A short message that pops in, e.g. "Image copied", and goes again after a
// couple of seconds. Use show() to show one; showing another while one is up
// replaces it and starts the time again. Where it goes is up to its parent,
// e.g. near the top of the window, in the middle.
//
// Carbon counts notifications as expressive moments, so it pops in with
// expressive entrance, then leaves with a quick productive exit.
Rectangle {
    id: root

    property alias text: label.text
    property bool shown: false
    // How long it stays up, in milliseconds.
    property int duration: 2000
    // Longer messages are cut short with "…".
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
