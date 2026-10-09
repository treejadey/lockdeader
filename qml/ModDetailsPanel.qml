import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects
import QtQuick.Layouts

// Shows one mod: a large preview with its name and author over it, the
// description, and an install button. While no mod is open it shows a hint
// instead, so the panel can stay on screen and the grid beside it never moves.
Rectangle {
    id: root

    // Whether a mod is open. While false, the last mod's details may still be
    // set; they're kept so they can fade out rather than vanish.
    property bool open: false
    property string name
    property string author
    property string image
    // All preview images, as {full, thumbnail, caption} objects.
    property var images: []
    property string profileUrl
    // Rich text (Qt's HTML subset).
    property string description
    property bool loading: false
    property string error: ""

    signal installClicked()
    // Emitted when one of the mod's images is clicked, to show it full size.
    signal imageClicked(int index)

    color: Theme.surface

    // Start each mod's description and gallery from the beginning.
    onProfileUrlChanged: {
        descriptionView.contentY = 0
        gallery.reset()
    }

    // The open mod's details.
    Item {
        id: content

        anchors.fill: parent
        opacity: 0
        visible: opacity > 0
        enabled: root.open
        transform: Translate {
            id: contentShift
            y: 8
        }

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Preview image with the name and author over it. Clicking it
            // shows the image full size.
            Item {
                Layout.fillWidth: true
                Layout.preferredHeight: Math.round(width * 1.15)
                clip: true

                Rectangle {
                    anchors.fill: parent
                    color: Theme.overlay
                }

                Image {
                    anchors.fill: parent
                    source: root.image
                    sourceSize: Qt.size(width, height)
                    asynchronous: true
                    fillMode: Image.PreserveAspectCrop
                    opacity: status === Image.Ready ? 1 : 0

                    Behavior on opacity {
                        NumberAnimation {
                            duration: Motion.fast02
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.productiveStandard
                        }
                    }
                }

                // Darkens the edges so the text and the link icon stay readable.
                Rectangle {
                    anchors.fill: parent
                    gradient: Gradient {
                        GradientStop { position: 0.0; color: Qt.rgba(0, 0, 0, 0.35) }
                        GradientStop { position: 0.4; color: Qt.rgba(0, 0, 0, 0) }
                        GradientStop { position: 1.0; color: Qt.rgba(0, 0, 0, 0.55) }
                    }
                }

                Item {
                    anchors {
                        left: parent.left
                        right: parent.right
                        bottom: parent.bottom
                        margins: 10
                    }
                    height: info.implicitHeight
                    layer.enabled: true
                    layer.effect: MultiEffect {
                        shadowEnabled: true
                        shadowColor: "black"
                        shadowOpacity: 0.6
                        shadowBlur: 1.0
                        blurMax: 4
                        shadowHorizontalOffset: 0
                        shadowVerticalOffset: 1
                    }

                    Column {
                        id: info

                        width: parent.width
                        spacing: 2

                        Label {
                            width: parent.width
                            text: root.name
                            color: "white"
                            font.pixelSize: 20
                            font.weight: Font.Medium
                            wrapMode: Text.Wrap
                            maximumLineCount: 3
                            elide: Text.ElideRight
                        }

                        Label {
                            width: parent.width
                            text: root.author
                            color: Theme.gold
                            font.pixelSize: 14
                            font.weight: Font.Medium
                            elide: Text.ElideRight
                        }
                    }
                }

                // MouseAreas rather than TapHandlers here and on the link icon
                // below: pointer handlers would both see a click on the icon.
                MouseArea {
                    anchors.fill: parent
                    enabled: root.images.length > 0
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.imageClicked(0)
                }

                // Opens the mod's GameBanana page in the browser.
                Image {
                    anchors {
                        top: parent.top
                        right: parent.right
                        margins: 8
                    }
                    width: 22
                    height: 22
                    source: "icons/open_24_regular.svg"
                    sourceSize: Qt.size(width, height)
                    visible: root.profileUrl !== ""
                    opacity: externalArea.containsMouse ? 1 : 0.8

                    MouseArea {
                        id: externalArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: Qt.openUrlExternally(root.profileUrl)
                    }

                    ToolTip.visible: externalArea.containsMouse
                    ToolTip.delay: 500
                    ToolTip.text: "Open on GameBanana"
                }
            }

            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                Flickable {
                    id: descriptionView

                    anchors.fill: parent
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    contentWidth: width
                    contentHeight: descriptionColumn.height + 20

                    ScrollBar.vertical: ScrollBar {
                        onPressedChanged: if (pressed) descriptionWheel.stop()
                    }

                    Column {
                        id: descriptionColumn

                        x: 10
                        y: 10
                        width: descriptionView.width - 20
                        spacing: 6

                        Label {
                            visible: gallery.visible
                            text: "Gallery"
                            color: Theme.subtle
                            font.bold: true
                        }

                        // Every image but the first, which is already the header
                        // above (and opens just the same when clicked).
                        GalleryCarousel {
                            id: gallery

                            width: parent.width
                            visible: root.images.length > 1
                            images: root.images.slice(1)
                            onActivated: (index) => root.imageClicked(index + 1)
                        }

                        Item {
                            width: 1
                            height: 4
                            visible: gallery.visible
                        }

                        Label {
                            text: "Description"
                            color: Theme.subtle
                            font.bold: true
                        }

                        BusyIndicator {
                            width: 32
                            height: 32
                            visible: running
                            running: root.loading && root.description === ""
                        }

                        Label {
                            width: parent.width
                            visible: root.error !== ""
                            text: "Couldn't load the description: " + root.error
                            color: Theme.love
                            wrapMode: Text.Wrap
                        }

                        Label {
                            width: parent.width
                            visible: text !== ""
                            text: root.description
                            textFormat: Text.RichText
                            wrapMode: Text.Wrap
                            color: Theme.text
                            linkColor: Theme.foam
                            onLinkActivated: (link) => Qt.openUrlExternally(link)

                            HoverHandler {
                                cursorShape: parent.hoveredLink !== "" ? Qt.PointingHandCursor : Qt.ArrowCursor
                            }
                        }
                    }
                }

                SmoothWheel {
                    id: descriptionWheel
                    anchors.fill: descriptionView
                    flickable: descriptionView
                }
            }

            // Opens the installation dialog, hence the "…".
            ActionButton {
                Layout.fillWidth: true
                Layout.margins: 6
                text: "Install mod…"
                icon.source: "icons/arrow_download_24_regular.svg"
                onClicked: root.installClicked()
            }
        }
    }

    // Shown while nothing is open.
    Column {
        id: hint

        anchors.centerIn: parent
        width: parent.width - 48
        spacing: 12

        Image {
            anchors.horizontalCenter: parent.horizontalCenter
            width: 40
            height: 40
            source: "icons/cursor_click_24_regular.svg"
            sourceSize: Qt.size(width, height)
            opacity: 0.5
        }

        Label {
            width: parent.width
            text: "Select a mod to see its details"
            color: Theme.subtle
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
        }
    }

    // Opening a mod is the moment to draw the eye, so its details come in with
    // Carbon's expressive entrance curve, rising slightly as they fade in.
    // Closing just gets out of the way: a quick productive exit.
    states: State {
        name: "open"
        when: root.open
        PropertyChanges {
            content.opacity: 1
            contentShift.y: 0
            hint.opacity: 0
        }
    }

    transitions: [
        Transition {
            to: "open"
            ParallelAnimation {
                NumberAnimation {
                    targets: [content, contentShift]
                    properties: "opacity,y"
                    duration: Motion.moderate01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.expressiveEntrance
                }
                NumberAnimation {
                    target: hint
                    property: "opacity"
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveExit
                }
            }
        },
        Transition {
            from: "open"
            ParallelAnimation {
                NumberAnimation {
                    targets: [content, contentShift]
                    properties: "opacity,y"
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveExit
                }
                NumberAnimation {
                    target: hint
                    property: "opacity"
                    duration: Motion.moderate01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveEntrance
                }
            }
        }
    ]
}
