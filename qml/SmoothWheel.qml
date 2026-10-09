import QtQuick

// Smooth, browser-like mouse wheel scrolling for a Flickable. Place it over the
// flickable (e.g. anchors.fill) so it sees wheel events first.
//
// Flickable turns every mouse wheel notch into its own short flick that slows
// to a stop before the next one, so continuous wheeling stutters. Here each
// notch moves a target position instead, and contentY eases towards it; notches
// that arrive mid-scroll extend the motion, so it keeps a steady pace. Touchpad
// scrolling (which has a pixel delta) is passed through to the flickable, which
// already handles it smoothly.
MouseArea {
    id: root

    required property Flickable flickable
    // How far one wheel notch scrolls, in pixels.
    property real step: 100

    // Stops any wheel scroll in progress, e.g. when the user grabs a scrollbar.
    function stop() {
        animation.stop()
    }

    function scrollBy(dy) {
        const f = root.flickable
        const top = f.originY - f.topMargin
        const bottom = Math.max(top, f.originY + f.contentHeight + f.bottomMargin - f.height)
        const start = animation.running ? animation.to : f.contentY

        animation.stop()
        animation.from = f.contentY
        animation.to = Math.max(top, Math.min(bottom, start + dy))
        animation.start()
    }

    acceptedButtons: Qt.NoButton
    // A MouseArea sets an arrow cursor by default, which would hide the cursor
    // of everything underneath (e.g. the pointing hand on cards).
    cursorShape: undefined

    onWheel: (wheel) => {
        if (wheel.pixelDelta.y !== 0 || wheel.angleDelta.y === 0) {
            wheel.accepted = false
            return
        }
        root.scrollBy(-wheel.angleDelta.y / 120 * root.step)
    }

    // Not a Carbon motion token: this animation is re-targeted on every notch
    // while the user keeps wheeling, and was tuned to keep that pace steady.
    NumberAnimation {
        id: animation
        target: root.flickable
        property: "contentY"
        duration: 200
        easing.type: Easing.OutQuad
    }

    // Dragging the content takes over from any wheel scroll in progress.
    Connections {
        target: root.flickable
        function onMovementStarted() {
            animation.stop()
        }
    }
}
