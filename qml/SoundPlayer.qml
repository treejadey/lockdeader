import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// The sound itself is played by the soundPreview bridge in Go, which plays one
// sound at a time. Any number of players can show the same sound: they all
// follow the bridge, so they stay in step. The volume is the bridge's too,
// shared by the whole app.
Rectangle {
    id: root

    property url source
    // Off where a VolumeControl elsewhere already sets the app's volume.
    property bool showVolume: true
    // For where the space above the bar is cut off.
    property bool bubbleBelow: false

    // From 0 to 1, for the whole app.
    readonly property real volume: soundPreview.volume
    readonly property bool muted: soundPreview.muted

    // Whether this player's sound is the one the bridge has loaded. Every
    // other player shows its sound as stopped.
    readonly property bool current: root.source.toString() !== ""
        && soundPreview.source === root.source.toString()
    readonly property bool playing: current && soundPreview.state === "playing"
    // Playing or paused partway; a stopped player is back at the start.
    readonly property bool started: current && soundPreview.state !== "stopped"
    readonly property bool failed: current && soundPreview.error !== ""
    // In milliseconds; 0 until the sound has been loaded once.
    readonly property int duration: current && soundPreview.duration > 0 ? soundPreview.duration
        : soundPreview.durations[root.source.toString()] || 0

    // From 0 to 1. While dragging, it follows the pointer; the sound only
    // seeks once it's let go.
    readonly property real progress: seekArea.pressed ? seekArea.ratio
        : started && duration > 0 ? soundPreview.position / duration
        : 0

    // Makes this player's sound the one the bridge plays, stopping any other.
    function load() {
        if (!root.current)
            soundPreview.source = root.source.toString()
    }

    function play() {
        root.load()
        soundPreview.state = "playing"
    }

    function pause() {
        if (root.playing)
            soundPreview.state = "paused"
    }

    // Also unmutes.
    function setVolume(volume) {
        soundPreview.volume = Math.round(Math.max(0, Math.min(1, volume)) * 100) / 100
        soundPreview.muted = false
    }

    function toggleMute() {
        if (root.muted || root.volume === 0) {
            // Unmuting at no volume would still be silent.
            if (root.volume === 0)
                soundPreview.volume = soundPreview.defaultVolume

            soundPreview.muted = false
        } else {
            soundPreview.muted = true
        }
    }

    function formatTime(ms) {
        const seconds = Math.floor(ms / 1000)

        return Math.floor(seconds / 60) + ":" + String(seconds % 60).padStart(2, "0")
    }

    implicitHeight: 36
    radius: height / 2
    color: Theme.overlay

    AbstractButton {
        id: playButton

        anchors {
            left: parent.left
            leftMargin: 6
            verticalCenter: parent.verticalCenter
        }
        width: 28
        height: 28
        padding: 5
        enabled: root.source.toString() !== "" && !root.failed
        hoverEnabled: true
        focusPolicy: Qt.TabFocus
        onClicked: root.playing ? root.pause() : root.play()

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
            color: playButton.hovered ? Theme.highlightMed : "transparent"
            border.width: playButton.visualFocus ? 1 : 0
            border.color: Theme.iris

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        // The pause icon is foam-coloured itself, to match the playing bar.
        contentItem: Image {
            source: root.playing ? "icons/pause_24_filled.svg" : "icons/play_24_regular.svg"
            sourceSize: Qt.size(width, height)
            fillMode: Image.PreserveAspectFit
            opacity: root.playing || playButton.hovered ? 1 : 0.6
        }
    }

    Label {
        id: time

        anchors {
            right: root.showVolume ? volumeButton.left : parent.right
            rightMargin: root.showVolume ? 8 : 12
            verticalCenter: parent.verticalCenter
        }
        // Monospaced, so the digits don't shift about as they change.
        font.family: app.monospaceFont
        font.pixelSize: 12
        text: root.failed ? qsTr("Can’t play")
            : root.formatTime(root.progress * root.duration) + "/"
              + (root.duration > 0 ? root.formatTime(root.duration) : "-:--")
        color: root.failed ? Theme.love : root.playing ? Theme.foam : Theme.subtle
    }

    // The whole height of the player responds to the pointer, not just the
    // thin bar, so it's easy to hit.
    Item {
        id: seekBar

        anchors {
            left: playButton.right
            leftMargin: 8
            right: time.left
            rightMargin: 12
            top: parent.top
            bottom: parent.bottom
        }

        // Thicker while pointed at, so it's clear it can be grabbed.
        readonly property bool highlighted: seekArea.containsMouse || seekArea.pressed

        Rectangle {
            id: track

            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            height: seekBar.highlighted ? 4 : 2
            radius: height / 2
            color: Theme.highlightHigh

            Behavior on height {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            // Lit up to the pointer, showing where a click would seek to.
            Rectangle {
                height: parent.height
                radius: parent.radius
                width: seekArea.ratio * parent.width
                color: Theme.subtle
                visible: seekArea.containsMouse && !seekArea.pressed
            }

            Rectangle {
                height: parent.height
                radius: parent.radius
                width: root.progress * parent.width
                color: Theme.foam
            }
        }

        // The playhead.
        Rectangle {
            x: root.progress * seekBar.width - width / 2
            anchors.verticalCenter: parent.verticalCenter
            width: 12
            height: 12
            radius: width / 2
            color: Theme.foam
            scale: root.started || seekBar.highlighted ? 1 : 0

            Behavior on scale {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

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
            id: seekArea

            // Where the pointer is along the bar, from 0 to 1.
            readonly property real ratio: Math.max(0, Math.min(1, mouseX / width))

            anchors.fill: parent
            enabled: root.duration > 0 && !root.failed
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onReleased: {
                // Seeking a stopped sound only leaves it paused there, but
                // clicking the bar of one that hasn't started should play it.
                const wasStarted = root.started

                root.load()
                soundPreview.seek = Math.round(ratio * root.duration)

                if (!wasStarted)
                    root.play()
            }
        }
    }

    // Kept within the player's width, so it never sticks out past its ends.
    Rectangle {
        id: bubble

        x: Math.max(0, Math.min(root.width - width,
                                seekBar.x + seekArea.ratio * seekBar.width - width / 2))
        y: root.bubbleBelow ? root.height + 6 : -height - 6
        width: bubbleText.implicitWidth + 12
        height: bubbleText.implicitHeight + 6
        radius: 4
        color: Theme.highlightMed
        opacity: seekBar.highlighted ? 1 : 0
        visible: opacity > 0

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        Label {
            id: bubbleText

            anchors.centerIn: parent
            font.family: app.monospaceFont
            font.pixelSize: 11
            text: root.formatTime(seekArea.ratio * root.duration)
            color: Theme.text
        }
    }

    // The volume slider rises out of the volume button and sinks back into it.
    // Declared before the button, so the button stays on top of it.
    //
    // It's clipped at the button's middle, not its bottom edge: the button is
    // round, so whatever slides behind its lower half would show through the
    // corners around it.
    Item {
        id: volumeSlider

        visible: root.showVolume

        // Stays open briefly after the pointer leaves, so it doesn't snap shut
        // when the pointer strays off for a moment.
        property bool shown: false
        readonly property bool wanted: volumeButton.hovered || volumeButton.activeFocus
            || (shown && sliderHover.hovered) || volumeArea.pressed

        onWantedChanged: {
            if (wanted) {
                closeDelay.stop()
                shown = true
            } else {
                closeDelay.restart()
            }
        }

        x: volumeButton.x
        y: volumeButton.y + volumeButton.height / 2 - height
        width: volumeButton.width
        // The panel's foot reaches below this, behind the button's lower half.
        height: panel.height - volumeButton.height / 2
        clip: true

        Timer {
            id: closeDelay

            interval: 200
            onTriggered: volumeSlider.shown = false
        }

        HoverHandler {
            id: sliderHover

            enabled: volumeSlider.shown
        }

        Rectangle {
            id: panel

            width: parent.width
            // Short enough to fit above the player even under a one-line name.
            height: 96
            radius: width / 2
            color: Theme.overlay
            // Below the clipping area, and transparent, while closed.
            y: volumeSlider.height
            opacity: 0

            Rectangle {
                id: volumeTrack

                anchors {
                    horizontalCenter: parent.horizontalCenter
                    top: parent.top
                    bottom: parent.bottom
                    topMargin: 12
                    // Clear of the button at the foot.
                    bottomMargin: volumeButton.height + 8
                }
                width: 4
                radius: width / 2
                color: Theme.highlightHigh

                Rectangle {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    height: volumeButton.level * parent.height
                    radius: parent.radius
                    color: Theme.foam
                }
            }

            // The handle.
            Rectangle {
                anchors.horizontalCenter: parent.horizontalCenter
                y: volumeTrack.y + (1 - volumeButton.level) * volumeTrack.height - height / 2
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
                id: volumeArea

                function setFromPointer() {
                    root.setVolume(1 - (mouseY - volumeTrack.y) / volumeTrack.height)
                }

                // Everything above the button.
                anchors {
                    fill: parent
                    bottomMargin: volumeButton.height
                }
                cursorShape: Qt.PointingHandCursor
                preventStealing: true
                onPressed: setFromPointer()
                onPositionChanged: if (pressed) setFromPointer()
                onWheel: (wheel) => root.setVolume(volumeButton.level + wheel.angleDelta.y / 120 * 0.05)
            }
        }

        // Carbon's productive motion, as for a dropdown: an entrance curve
        // opening, and a quicker exit closing.
        states: State {
            name: "shown"
            when: volumeSlider.shown

            PropertyChanges {
                panel.y: 0
                panel.opacity: 1
            }
        }

        transitions: [
            Transition {
                to: "shown"

                NumberAnimation {
                    target: panel
                    properties: "y,opacity"
                    duration: Motion.moderate01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveEntrance
                }
            },
            Transition {
                from: "shown"

                NumberAnimation {
                    target: panel
                    properties: "y,opacity"
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveExit
                }
            }
        ]
    }

    AbstractButton {
        id: volumeButton

        // The volume as heard, from 0 to 1.
        readonly property real level: root.muted ? 0 : root.volume

        anchors {
            right: parent.right
            rightMargin: 4
            verticalCenter: parent.verticalCenter
        }
        visible: root.showVolume
        width: 28
        height: 28
        padding: 5
        hoverEnabled: true
        focusPolicy: Qt.TabFocus
        onClicked: root.toggleMute()
        Keys.onUpPressed: root.setVolume(root.volume + 0.05)
        Keys.onDownPressed: root.setVolume(root.volume - 0.05)

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

        WheelHandler {
            onWheel: (event) => root.setVolume(volumeButton.level + event.angleDelta.y / 120 * 0.05)
        }

        background: Rectangle {
            radius: height / 2
            color: volumeButton.hovered || volumeSlider.shown ? Theme.highlightMed : Theme.highlightLow
            border.width: volumeButton.visualFocus ? 1 : 0
            border.color: Theme.iris

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        contentItem: Image {
            readonly property string speaker: volumeButton.level === 0 ? "speaker_mute"
                : volumeButton.level < 0.34 ? "speaker_0"
                : volumeButton.level < 0.67 ? "speaker_1"
                : "speaker_2"

            source: "icons/" + speaker + (volumeSlider.shown ? "_24_filled.svg" : "_24_regular.svg")
            sourceSize: Qt.size(width, height)
            fillMode: Image.PreserveAspectFit
            opacity: volumeButton.hovered || volumeSlider.shown ? 1 : 0.6
        }
    }
}
