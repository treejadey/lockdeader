import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects
import QtQuick.Layouts

// While no mod is open, the panel shows a hint instead, so it can stay on
// screen and the grid beside it never moves.
Rectangle {
    id: root

    // Whether a mod is open. While false, the last mod's details may still be
    // set; they're kept so they can fade out rather than vanish.
    property bool open: false
    property string name
    property string author
    property string authorUrl
    property string image
    // All preview images, as {full, thumbnail, caption} objects.
    property var images: []
    property string audio
    property string profileUrl
    // Rich text (Qt's HTML subset).
    property string description
    property bool loading: false
    property string error: ""
    property string errorDetail: ""
    // Not needed where the sound's player stays on screen elsewhere, e.g. in
    // the list of sounds.
    property bool pauseOnClose: true

    signal installClicked()
    signal imageClicked(int index)

    color: Theme.surface

    onOpenChanged: if (!open && pauseOnClose) player.pause()

    // Start each mod's description and gallery from the beginning.
    onProfileUrlChanged: {
        descriptionView.contentY = 0
        gallery.reset()
    }

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

            Item {
                id: header

                readonly property bool hasImage: root.image !== ""

                Layout.fillWidth: true
                Layout.preferredHeight: hasImage ? Math.round(width * 1.15) : info.implicitHeight + 20
                clip: true

                Rectangle {
                    anchors.fill: parent
                    visible: header.hasImage
                    color: Theme.overlay
                }

                Image {
                    anchors.fill: parent
                    visible: header.hasImage
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
                    visible: header.hasImage
                    gradient: Gradient {
                        GradientStop { position: 0.0; color: Qt.rgba(0, 0, 0, 0.35) }
                        GradientStop { position: 0.4; color: Qt.rgba(0, 0, 0, 0) }
                        GradientStop { position: 1.0; color: Qt.rgba(0, 0, 0, 0.55) }
                    }
                }

                // MouseAreas rather than TapHandlers here, over the text and on
                // the link icon below: pointer handlers would all see a click
                // on the text or the icon. Declared before the text, so the
                // text is on top of it.
                MouseArea {
                    anchors.fill: parent
                    enabled: root.images.length > 0
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.imageClicked(0)
                }

                TextShadow {
                    source: headerText
                }

                Item {
                    id: headerText

                    anchors {
                        left: parent.left
                        right: parent.right
                        bottom: parent.bottom
                        margins: 10
                        // Without an image, the name starts at the top, beside the link icon.
                        rightMargin: header.hasImage ? 10 : 40
                    }
                    height: info.implicitHeight

                    Column {
                        id: info

                        width: parent.width
                        spacing: 2

                        Label {
                            width: parent.width
                            text: root.name
                            color: Theme.textOnPicture
                            font.pixelSize: 20
                            font.weight: Font.Medium
                            wrapMode: Text.Wrap
                            maximumLineCount: 3
                            elide: Text.ElideRight
                        }

                        Label {
                            // Only as wide as the name, so only the name is a link.
                            width: Math.min(implicitWidth, parent.width)
                            text: root.author
                            color: authorArea.containsMouse ? Theme.lift(Theme.gold, 0.15) : Theme.gold
                            font.pixelSize: 14
                            font.weight: Font.Medium
                            font.underline: root.authorUrl !== ""
                            elide: Text.ElideRight

                            Behavior on color {
                                ColorAnimation {
                                    duration: Motion.fast01
                                    easing.type: Easing.BezierSpline
                                    easing.bezierCurve: Motion.productiveStandard
                                }
                            }

                            MouseArea {
                                id: authorArea

                                anchors.fill: parent
                                enabled: root.authorUrl !== ""
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: Qt.openUrlExternally(root.authorUrl)
                            }

                            ToolTip.visible: authorArea.containsMouse
                            ToolTip.delay: 500
                            ToolTip.text: qsTr("Open profile on GameBanana")
                        }
                    }

                    // The name isn't a link: this keeps the image's pointer and
                    // click away from it. Under the author's link.
                    MouseArea {
                        anchors.fill: info
                        z: -1
                        hoverEnabled: true
                    }
                }

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
                    ToolTip.text: qsTr("Open on GameBanana")
                }
            }

            SoundPlayer {
                id: player

                Layout.fillWidth: true
                Layout.leftMargin: 10
                Layout.rightMargin: 10
                Layout.topMargin: 2
                visible: root.audio !== ""
                source: root.audio
                // The volume is in the bar along the top.
                showVolume: false
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
                            text: qsTr("Gallery")
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
                            text: qsTr("Description")
                            color: Theme.subtle
                            font.bold: true
                        }

                        BusyIndicator {
                            width: 32
                            height: 32
                            visible: running
                            running: root.loading && root.description === ""
                        }

                        ErrorLabel {
                            width: parent.width
                            visible: root.error !== ""
                            text: qsTr("Couldn’t load the description: %1").arg(root.error)
                            detail: root.errorDetail
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
                text: qsTr("Install Mod…")
                icon.source: "icons/arrow_download_24_regular.svg"
                onClicked: root.installClicked()
            }
        }
    }

    Column {
        id: hint

        anchors.centerIn: parent
        width: parent.width - 48
        spacing: 12

        TintedIcon {
            anchors.horizontalCenter: parent.horizontalCenter
            width: 40
            height: 40
            source: "icons/cursor_click_24_regular.svg"
            opacity: 0.5
        }

        Label {
            width: parent.width
            text: qsTr("Select a mod to see its details")
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
