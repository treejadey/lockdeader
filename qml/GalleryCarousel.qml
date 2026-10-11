import QtQuick

Item {
    id: root

    // {full, thumbnail, caption} objects.
    property var images: []

    signal activated(int index)

    function reset() {
        scroll.stop()
        strip.positionViewAtBeginning()
    }

    // Scrolls by about a row's width, keeping one thumbnail from before in view.
    function page(direction) {
        const minX = strip.originX
        const maxX = Math.max(minX, strip.originX + strip.contentWidth - strip.width)
        // Carry on from where a scroll that's still running is headed, so
        // quick clicks add up.
        const start = scroll.running ? scroll.to : strip.contentX
        const distance = Math.max(strip.thumbnailWidth, strip.width - strip.thumbnailWidth - strip.spacing)

        scroll.stop()
        scroll.from = strip.contentX
        scroll.to = Math.max(minX, Math.min(maxX, start + direction * distance))
        scroll.start()
    }

    implicitHeight: strip.implicitHeight

    GalleryStrip {
        id: strip

        anchors.fill: parent
        images: root.images
        onActivated: (index) => root.activated(index)
        onMovementStarted: scroll.stop()
    }

    NumberAnimation {
        id: scroll

        target: strip
        property: "contentX"
        duration: Motion.moderate02
        easing.type: Easing.BezierSpline
        easing.bezierCurve: Motion.productiveStandard
    }

    ScrollArrow {
        anchors.left: parent.left
        shown: !strip.atXBeginning
        icon.source: "icons/chevron_left_24_regular.svg"
        onClicked: root.page(-1)
    }

    ScrollArrow {
        anchors.right: parent.right
        shown: !strip.atXEnd
        icon.source: "icons/chevron_right_24_regular.svg"
        onClicked: root.page(1)
    }

    component ScrollArrow: ActionButton {
        property bool shown: false

        anchors {
            verticalCenter: parent.verticalCenter
            margins: 4
        }
        implicitWidth: 28
        implicitHeight: 28
        leftPadding: 0
        rightPadding: 0
        opacity: shown ? 1 : 0
        visible: opacity > 0
        enabled: shown

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }
}
