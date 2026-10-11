import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// The app's volume for sound previews: a speaker button and a horizontal
// slider in a pill. Clicking the speaker mutes and unmutes; clicking or
// dragging the slider sets the volume, and scrolling over either, or the arrow
// keys while the speaker has focus, change it. The volume belongs to the
// soundPreview bridge, which saves it in the settings.
Rectangle {
    id: root

    // The volume as heard, from 0 to 1.
    readonly property real level: soundPreview.muted ? 0 : soundPreview.volume

    // Sets the volume, which also unmutes.
    function setVolume(volume) {
        soundPreview.volume = Math.round(Math.max(0, Math.min(1, volume)) * 100) / 100
        soundPreview.muted = false
    }

    function toggleMute() {
        if (soundPreview.muted || soundPreview.volume === 0) {
            // Unmuting at no volume would still be silent.
            if (soundPreview.volume === 0)
                soundPreview.volume = soundPreview.defaultVolume
            soundPreview.muted = false
        } else {
            soundPreview.muted = true
        }
    }

    implicitWidth: 132
    implicitHeight: 36
    radius: height / 2
    color: Theme.overlay

    WheelHandler {
        onWheel: (event) => root.setVolume(root.level + event.angleDelta.y / 120 * 0.05)
    }

    AbstractButton {
        id: speaker

        anchors {
            left: parent.left
            leftMargin: 4
            verticalCenter: parent.verticalCenter
        }
        width: 28
        height: 28
        padding: 5
        hoverEnabled: true
        focusPolicy: Qt.TabFocus
        onClicked: root.toggleMute()
        Keys.onUpPressed: root.setVolume(soundPreview.volume + 0.05)
        Keys.onRightPressed: root.setVolume(soundPreview.volume + 0.05)
        Keys.onDownPressed: root.setVolume(soundPreview.volume - 0.05)
        Keys.onLeftPressed: root.setVolume(soundPreview.volume - 0.05)

        scale: down ? 0.92 : 1

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
            radius: height / 2
            color: speaker.hovered ? Theme.highlightMed : "transparent"
            // Only shown when focused with the keyboard.
            border.width: speaker.visualFocus ? 1 : 0
            border.color: Theme.iris

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        // The speaker's waves show the volume. Filled while it's being set.
        contentItem: Image {
            readonly property string icon: root.level === 0 ? "speaker_mute"
                : root.level < 0.34 ? "speaker_0"
                : root.level < 0.67 ? "speaker_1"
                : "speaker_2"

            source: "icons/" + icon + (sliderArea.pressed ? "_24_filled.svg" : "_24_regular.svg")
            sourceSize: Qt.size(width, height)
            fillMode: Image.PreserveAspectFit
            opacity: speaker.hovered || sliderArea.pressed ? 1 : 0.6
        }
    }

    Item {
        id: slider

        anchors {
            left: speaker.right
            leftMargin: 8
            right: parent.right
            rightMargin: 14
            top: parent.top
            bottom: parent.bottom
        }

        // Thicker while pointed at, so it's clear it can be grabbed, like the
        // sound player's seek bar.
        readonly property bool highlighted: sliderArea.containsMouse || sliderArea.pressed

        Rectangle {
            id: track

            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            height: slider.highlighted ? 4 : 2
            radius: height / 2
            color: Theme.highlightHigh

            Behavior on height {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            Rectangle {
                height: parent.height
                radius: parent.radius
                width: root.level * parent.width
                color: Theme.foam
            }
        }

        // The handle, with a soft shadow under it, like the playhead.
        Rectangle {
            x: root.level * slider.width - width / 2
            anchors.verticalCenter: parent.verticalCenter
            width: 12
            height: 12
            radius: width / 2
            color: Theme.foam

            layer.enabled: true
            layer.effect: MultiEffect {
                shadowEnabled: true
                shadowColor: "black"
                shadowOpacity: 0.5
                shadowBlur: 0.3
                shadowVerticalOffset: 1
                shadowHorizontalOffset: 0
            }
        }

        MouseArea {
            id: sliderArea

            function setFromPointer() {
                root.setVolume((mouseX + anchors.leftMargin) / slider.width)
            }

            // A little past the ends, so the handle's edges can be grabbed too.
            anchors {
                fill: parent
                leftMargin: -6
                rightMargin: -6
            }
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            preventStealing: true
            onPressed: setFromPointer()
            onPositionChanged: if (pressed) setFromPointer()
        }
    }
}
