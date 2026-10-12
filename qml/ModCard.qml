import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// The mod's picture, with its name and when it was updated underneath.
// Items declared in a ModCard go on the picture, over everything else (e.g. a
// checkbox in a corner), and get their own clicks.
Item {
    id: card

    required property int modId
    required property string name
    // Unix timestamp of the last update, in seconds.
    required property double updated
    required property string thumbnail
    // GameBanana shows the mod behind a warning or hides it (see banana.Mod.IsNSFW).
    required property bool nsfw

    property int tileWidth: 240
    property int tileHeight: 135
    // Room under the picture for the text. Fixed, so cells don't depend on
    // what's in them.
    property int textHeight: 48
    property int radius: 14
    // Selected cards keep the hover overlay, so the open mod stays marked.
    property bool selected: false
    property real horizontalOffset: 0
    property bool showUpdated: true
    // From GameBanana; -1 means they aren't known, and hides them.
    property int views: -1
    property int likes: -1
    property bool dimmed: false
    readonly property alias hovered: hover.hovered

    default property alias tileContent: tile.data

    signal clicked()
    // Emitted on a right click.
    signal menuRequested()

    // The picture and the text under it, which are clicked as one.
    Item {
        anchors.centerIn: parent
        anchors.horizontalCenterOffset: card.horizontalOffset
        width: card.tileWidth
        height: card.tileHeight + card.textHeight

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

        Item {
            id: tile

            width: card.tileWidth
            height: card.tileHeight

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
                    // Fills the tile, cut from the top when the picture is
                    // taller, as GameBanana's own cards do: that's where
                    // the subject usually is.
                    fillMode: Image.PreserveAspectCrop
                    verticalAlignment: Image.AlignTop
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
        }

        Item {
            id: cardText

            anchors {
                top: tile.bottom
                topMargin: 8
                left: parent.left
                right: parent.right
                bottom: parent.bottom
            }

            Label {
                id: nameLabel

                anchors {
                    top: parent.top
                    left: parent.left
                    right: parent.right
                }
                text: card.name
                color: card.dimmed ? Theme.subtle : Theme.text
                font.pixelSize: 15
                elide: Text.ElideRight

                HoverHandler {
                    id: nameHover
                }

                // The whole name, when it's cut short.
                ToolTip {
                    visible: nameHover.hovered && nameLabel.truncated
                    delay: 500
                    text: nameLabel.text
                    width: Math.min(implicitWidth, 360)
                }
            }

            Row {
                anchors {
                    top: nameLabel.bottom
                    topMargin: 2
                    left: parent.left
                    right: stats.visible ? stats.left : parent.right
                    rightMargin: stats.visible ? 8 : 0
                }
                visible: card.showUpdated
                spacing: 4

                TintedIcon {
                    anchors.verticalCenter: parent.verticalCenter
                    width: 14
                    height: 14
                    source: "icons/calendar_ltr_24_regular.svg"
                    color: dateLabel.color
                }

                Label {
                    id: dateLabel

                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - 14 - parent.spacing
                    text: RelativeTime.format(card.updated, RelativeTime.now)
                    color: card.dimmed ? Theme.muted : Theme.subtle
                    font.pixelSize: 13
                    elide: Text.ElideRight
                }
            }

            // On the date's line, against the card's right edge.
            ModStats {
                id: stats

                anchors {
                    top: nameLabel.bottom
                    topMargin: 2
                    right: parent.right
                }
                visible: card.views >= 0 && card.likes >= 0
                views: card.views
                likes: card.likes
                color: dateLabel.color
            }
        }
    }
}
