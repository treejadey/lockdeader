import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// A single tile in the mod grid: a square, rounded preview image with the mod's
// name in the top-left corner, and when it was last updated and its author in
// the bottom-left. NSFW mods get a thin red border and a heavily blurred
// preview; the details panel shows their images as normal.
// The tile is centred in whatever cell the grid gives it. Items declared in a
// ModCard go on the tile, over everything else, e.g. a checkbox in a corner;
// they get their own clicks.
Item {
    id: card

    required property int modId
    required property string name
    required property string author
    // Unix timestamp of the last update, in seconds.
    required property double updated
    required property string thumbnail
    // GameBanana shows the mod behind a warning or hides it (see banana.Mod.IsNSFW).
    required property bool nsfw

    property int tileSize: 150
    property int radius: 14
    // Selected cards keep the hover overlay, so the open mod stays marked.
    property bool selected: false
    // Moves the tile sideways from the centre of its cell.
    property real horizontalOffset: 0
    property bool showUpdated: true
    property bool showAuthor: true
    // Darkens the image, e.g. while the mod is installing.
    property bool dimmed: false
    // Keeps the bottom line clear of whatever is in the tile's bottom-right
    // corner.
    property real cornerRightMargin: 0
    readonly property alias hovered: hover.hovered

    default property alias tileContent: tile.data

    signal clicked()
    // Emitted on a right click.
    signal menuRequested()

    Item {
        id: tile

        anchors.centerIn: parent
        anchors.horizontalCenterOffset: card.horizontalOffset
        width: card.tileSize
        height: card.tileSize

        HoverHandler {
            id: hover
            cursorShape: Qt.PointingHandCursor
        }

        // A MouseArea rather than a TapHandler, so clicks on what's added to
        // the tile, which is drawn over it, don't reach it.
        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.LeftButton | Qt.RightButton
            onClicked: (mouse) => mouse.button === Qt.RightButton ? card.menuRequested() : card.clicked()
        }

        // Image and hover overlay, clipped to rounded corners by the mask below.
        Item {
            anchors.fill: parent
            layer.enabled: true
            layer.effect: MultiEffect {
                maskEnabled: true
                maskSource: mask
                // Soften the mask edge so the corners are antialiased.
                maskThresholdMin: 0.5
                maskSpreadAtMin: 1.0
            }

            // Shown until the image arrives; the tile's size never depends on it.
            Rectangle {
                anchors.fill: parent
                color: Theme.overlay
            }

            Image {
                anchors.fill: parent
                source: card.thumbnail
                sourceSize: Qt.size(width, height)
                asynchronous: true
                fillMode: Image.PreserveAspectCrop
                opacity: status === Image.Ready ? 1 : 0

                layer.enabled: card.nsfw
                layer.effect: MultiEffect {
                    blurEnabled: true
                    blur: 1.0
                    blurMax: 64
                    // Without padding, the blur repeats the edge pixels instead
                    // of fading into transparency around the border.
                    autoPaddingEnabled: false
                }

                Behavior on opacity {
                    NumberAnimation {
                        duration: Motion.fast02
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            Rectangle {
                anchors.fill: parent
                color: Theme.base
                opacity: card.dimmed ? 0.55 : 0

                Behavior on opacity {
                    NumberAnimation {
                        duration: Motion.moderate01
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            // Darkens the top and bottom of the hovered card, so it's clear
            // which one the pointer is over.
            Rectangle {
                anchors.fill: parent
                opacity: hover.hovered || card.selected ? 1 : 0
                gradient: Gradient {
                    GradientStop { position: 0.0; color: Qt.rgba(0, 0, 0, 0.35) }
                    GradientStop { position: 0.5; color: Qt.rgba(0, 0, 0, 0) }
                    GradientStop { position: 1.0; color: Qt.rgba(0, 0, 0, 0.35) }
                }

                Behavior on opacity {
                    NumberAnimation {
                        duration: Motion.fast02
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }
        }

        Item {
            id: mask

            anchors.fill: parent
            layer.enabled: true
            visible: false

            Rectangle {
                anchors.fill: parent
                radius: card.radius
                color: "black"
            }
        }

        // Marks NSFW mods. The gold outline below covers it while selected.
        Rectangle {
            anchors.fill: parent
            visible: card.nsfw
            radius: card.radius
            color: "transparent"
            border.color: Theme.love
            border.width: 1
        }

        // Marks the mod that's open in the details panel.
        Rectangle {
            anchors.fill: parent
            visible: card.selected
            radius: card.radius
            color: "transparent"
            border.color: Theme.gold
            border.width: 2
        }

        // All text gets a small, soft shadow so it stays readable on bright images.
        TextShadow {
            source: cardText
            opacity: 0.75
        }

        Item {
            id: cardText

            anchors.fill: parent
            anchors.margins: 10

            Label {
                anchors {
                    top: parent.top
                    left: parent.left
                    right: parent.right
                }
                text: card.name
                color: "white"
                font.pixelSize: 13
                wrapMode: Text.Wrap
                maximumLineCount: 2
                elide: Text.ElideRight
            }

            // When the mod was last updated, e.g. "3 hours ago".
            Row {
                anchors {
                    bottom: author.visible ? author.top : parent.bottom
                    left: parent.left
                    right: parent.right
                    // Only the bottom line is beside the corner.
                    rightMargin: author.visible ? 0 : card.cornerRightMargin
                }
                visible: card.showUpdated
                spacing: 4

                Image {
                    anchors.verticalCenter: parent.verticalCenter
                    width: 14
                    height: 14
                    source: "icons/calendar_ltr_24_regular.svg"
                    sourceSize: Qt.size(width, height)
                }

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - 14 - parent.spacing
                    text: RelativeTime.format(card.updated, RelativeTime.now)
                    color: "white"
                    font.pixelSize: 12
                    font.weight: Font.DemiBold
                    elide: Text.ElideRight
                }
            }

            Label {
                id: author

                anchors {
                    bottom: parent.bottom
                    left: parent.left
                    right: parent.right
                    rightMargin: card.cornerRightMargin
                }
                visible: card.showAuthor
                text: card.author
                color: Theme.gold
                font.pixelSize: 12
                elide: Text.ElideRight
            }
        }
    }
}
