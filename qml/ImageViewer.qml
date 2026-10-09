import QtQuick
import QtQuick.Controls.Fusion

// A full-window image viewer, much like Discord's. The image is shown as large
// as fits, but never larger than its own size, over a dimmed backdrop. Below it
// is its caption and, for galleries, a strip of all the images.
//
// Left/Right or the arrow buttons step through the images. Escape, the close
// button or clicking the backdrop closes it. Use show() to open it.
//
// Motion: opening a big image is a moment worth noticing, so the image comes
// in with Carbon's expressive entrance, growing slightly into place, while the
// backdrop dims over Carbon's background dimming duration. Closing is a quick
// productive exit.
Popup {
    id: root

    // {full, thumbnail, caption} objects, as in the details panel.
    property var images: []
    property int index: 0
    readonly property var current: images.length > 0
        ? images[Math.max(0, Math.min(index, images.length - 1))]
        : ({ full: "", thumbnail: "", caption: "" })

    function show(images, index) {
        root.images = images
        root.index = index
        open()
    }

    function step(delta) {
        index = Math.max(0, Math.min(images.length - 1, index + delta))
    }

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
                target: stage
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }
            NumberAnimation {
                target: stage
                property: "scale"
                from: 0.96
                to: 1
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }
        }
    }

    exit: Transition {
        NumberAnimation {
            targets: [backdrop, stage]
            property: "opacity"
            to: 0
            duration: Motion.moderate01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveExit
        }
    }

    Shortcut {
        sequence: "Left"
        enabled: root.opened
        onActivated: root.step(-1)
    }

    Shortcut {
        sequence: "Right"
        enabled: root.opened
        onActivated: root.step(1)
    }

    Rectangle {
        id: backdrop

        anchors.fill: parent
        color: Qt.rgba(Theme.base.r, Theme.base.g, Theme.base.b, 0.92)

        // A MouseArea rather than a TapHandler: pointer handlers see every
        // press under them, even ones the image above has already taken.
        MouseArea {
            anchors.fill: parent
            onClicked: root.close()
        }
    }

    Item {
        id: stage

        anchors.fill: parent

        // The space the image may take up.
        Item {
            id: area

            anchors {
                top: parent.top
                bottom: caption.top
                left: parent.left
                right: parent.right
                topMargin: 64
                bottomMargin: 12
                leftMargin: 72
                rightMargin: 72
            }
        }

        Item {
            id: frame

            // Until the full image arrives, its thumbnail stands in at the
            // same aspect ratio, so the frame is already the right shape.
            readonly property bool fullReady: full.status === Image.Ready
            readonly property real aspect: fullReady ? full.implicitWidth / full.implicitHeight
                : placeholder.status === Image.Ready ? placeholder.implicitWidth / placeholder.implicitHeight
                : 16 / 9
            readonly property real maxWidth: fullReady ? Math.min(area.width, full.implicitWidth) : area.width
            readonly property real maxHeight: fullReady ? Math.min(area.height, full.implicitHeight) : area.height

            anchors.centerIn: area
            width: Math.max(0, Math.min(maxWidth, maxHeight * aspect))
            height: width / aspect

            Image {
                id: placeholder
                anchors.fill: parent
                source: root.current.thumbnail
                asynchronous: true
            }

            Image {
                id: full

                anchors.fill: parent
                source: root.current.full
                asynchronous: true
                opacity: status === Image.Ready ? 1 : 0

                Behavior on opacity {
                    NumberAnimation {
                        duration: Motion.fast02
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            BusyIndicator {
                anchors.centerIn: parent
                running: full.status === Image.Loading
            }

            // Clicking the image itself shouldn't close the viewer.
            MouseArea {
                anchors.fill: parent
            }
        }

        Label {
            id: caption

            anchors {
                bottom: strip.visible ? strip.top : parent.bottom
                bottomMargin: strip.visible ? 16 : 24
                horizontalCenter: parent.horizontalCenter
            }
            width: Math.min(implicitWidth, parent.width - 48)
            height: text !== "" ? implicitHeight : 0
            text: root.current.caption
            color: Theme.text
            elide: Text.ElideRight
        }

        GalleryStrip {
            id: strip

            anchors {
                bottom: parent.bottom
                bottomMargin: 24
                horizontalCenter: parent.horizontalCenter
            }
            width: Math.min(contentWidth, parent.width - 48)
            thumbnailWidth: 96
            thumbnailHeight: 64
            visible: root.images.length > 1
            images: root.images
            selectedIndex: root.index
            onActivated: (index) => root.index = index
        }

        ActionButton {
            anchors {
                left: parent.left
                leftMargin: 16
                verticalCenter: area.verticalCenter
            }
            implicitWidth: 40
            implicitHeight: 40
            icon.source: "icons/chevron_left_24_regular.svg"
            icon.width: 24
            icon.height: 24
            visible: root.images.length > 1
            enabled: root.index > 0
            onClicked: root.step(-1)
        }

        ActionButton {
            anchors {
                right: parent.right
                rightMargin: 16
                verticalCenter: area.verticalCenter
            }
            implicitWidth: 40
            implicitHeight: 40
            icon.source: "icons/chevron_right_24_regular.svg"
            icon.width: 24
            icon.height: 24
            visible: root.images.length > 1
            enabled: root.index < root.images.length - 1
            onClicked: root.step(1)
        }

        ActionButton {
            anchors {
                top: parent.top
                right: parent.right
                margins: 16
            }
            implicitWidth: implicitHeight
            icon.source: "icons/dismiss_24_regular.svg"
            onClicked: root.close()

            ToolTip.visible: hovered
            ToolTip.delay: 500
            ToolTip.text: "Close (Esc)"
        }
    }
}
