import QtQuick
import QtQuick.Controls.Fusion

AbstractButton {
    id: root

    readonly property int knobSize: 24
    readonly property int inset: 3

    function flip() {
        checked = !checked
        toggled()
    }

    // amount is from 0 to 1.
    function mix(a, b, amount) {
        return Qt.rgba(a.r + (b.r - a.r) * amount, a.g + (b.g - a.g) * amount,
                       a.b + (b.b - a.b) * amount, a.a + (b.a - a.a) * amount)
    }

    implicitWidth: 2 * knobSize + 2 * inset
    implicitHeight: knobSize + 2 * inset

    hoverEnabled: true
    // Reachable with Tab, but clicking doesn't take focus from where it was.
    focusPolicy: Qt.TabFocus

    // Flipped by hand on press, rather than by AbstractButton on release.
    checkable: false
    onPressed: flip()

    // Where the knob is, from 0 (off) to 1 (on).
    QtObject {
        id: knob

        property real position: 0
    }

    // Starts where it is, without sliding there.
    Component.onCompleted: {
        slide.stop()
        knob.position = checked ? 1 : 0
    }

    onCheckedChanged: {
        slide.to = checked ? 1 : 0
        slide.restart()
    }

    // The pill changes colour straight away, then the knob slides over,
    // taking on its new colour as it goes, so it reads as the knob carrying
    // the change across.
    SequentialAnimation {
        id: slide

        property alias to: travel.to

        PauseAnimation {
            duration: Motion.fast01
        }

        NumberAnimation {
            id: travel

            target: knob
            property: "position"
            duration: Motion.moderate01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }

    background: Rectangle {
        radius: height / 2
        // Foam, darkened to sit behind a foam knob.
        color: root.checked
            ? Qt.hsva(Theme.foam.hsvHue, Math.min(1, Theme.foam.hsvSaturation * 1.45), Theme.foam.hsvValue * 0.5, 1)
            : root.hovered ? Theme.highlightHigh : Theme.highlightMed

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        // The keyboard focus ring.
        Rectangle {
            anchors.fill: parent
            anchors.margins: -3
            radius: height / 2
            color: "transparent"
            visible: root.visualFocus
            border.width: 1
            border.color: Theme.iris
        }
    }

    contentItem: Item {
        Rectangle {
            x: root.inset + knob.position * (root.width - root.knobSize - 2 * root.inset)
            anchors.verticalCenter: parent.verticalCenter
            width: root.knobSize
            height: root.knobSize
            radius: width / 2
            color: root.mix(root.hovered ? Theme.text : Theme.subtle, Theme.foam, knob.position)
            scale: root.down ? 0.92 : 1

            Behavior on scale {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
    }
}
