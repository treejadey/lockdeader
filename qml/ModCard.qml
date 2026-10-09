import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// A single tile in the mod grid: a square, rounded preview image with the mod's
// name in the top-left corner, and when it was last updated and its author in
// the bottom-left.
// The tile is centred in whatever cell the grid gives it.
Item {
    id: card

    required property int modId
    required property string name
    required property string author
    // Unix timestamp of the last update, in seconds.
    required property double updated
    required property string thumbnail

    property int tileSize: 150
    property int radius: 14
    // Selected cards keep the hover overlay, so the open mod stays marked.
    property bool selected: false
    // Moves the tile sideways from the centre of its cell.
    property real horizontalOffset: 0

    signal clicked()

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

        TapHandler {
            onTapped: card.clicked()
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

                Behavior on opacity {
                    NumberAnimation {
                        duration: Motion.fast02
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
        Item {
            anchors.fill: parent
            anchors.margins: 10
            layer.enabled: true
            layer.effect: MultiEffect {
                shadowEnabled: true
                shadowColor: "black"
                shadowOpacity: 0.75
                shadowBlur: 1.0
                blurMax: 4
                shadowHorizontalOffset: 0
                shadowVerticalOffset: 1
            }

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
                    bottom: author.top
                    left: parent.left
                    right: parent.right
                }
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
                }
                text: card.author
                color: Theme.gold
                font.pixelSize: 12
                elide: Text.ElideRight
            }
        }
    }
}
