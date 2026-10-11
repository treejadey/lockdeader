import QtQuick
import QtQuick.Controls.Fusion

// Mods as a vertically scrolling list, one row each: a small preview image
// and the name, the category, the author and when it was last updated. Sounds
// have no preview image, so they're always listed this way, and get a player
// for their audio preview in place of the image and the category. Clicking a
// row, anywhere but its player, opens it.
//
// The name always keeps some room. As the list narrows, the category (or the
// player) shrinks first, then the category goes, then when it was updated.
Item {
    id: root

    // Lists sounds rather than mods.
    property bool sounds: false

    property alias model: list.model
    readonly property alias count: list.count
    // Shown at the bottom of the list while the next page is on its way.
    property bool loading: false
    property string error: ""
    // The row with this mod id is drawn as selected; 0 for none.
    property int selectedModId: 0

    // The columns, from the left: the image, the name, the category (or for
    // sounds, the player), the author and when it was updated. Each column's
    // width includes the gap after it.
    readonly property int leadingWidth: sounds ? 12 : 64
    readonly property int nameMinWidth: 140
    readonly property int middleGap: sounds ? 8 : 22
    readonly property int middleMinWidth: sounds ? 180 : 120
    readonly property int middleMaxWidth: sounds ? 250 : 238
    readonly property int authorWidth: 120 + 12
    readonly property int updatedWidth: 116 + 12
    // The player always shows, so it counts here; the category doesn't.
    readonly property bool showUpdated: list.width - leadingWidth - nameMinWidth
        - (sounds ? middleGap + middleMinWidth : 0) - authorWidth - updatedWidth >= 0
    readonly property real middleWidth: {
        const room = list.width - leadingWidth - nameMinWidth - middleGap - authorWidth
            - (showUpdated ? updatedWidth : 12)
        return sounds ? Math.max(middleMinWidth, Math.min(middleMaxWidth, room))
                      : Math.min(middleMaxWidth, room)
    }
    readonly property bool showCategory: !sounds && middleWidth >= middleMinWidth

    // Emitted when the user asks to retry after a failed page load.
    signal retry()
    signal modClicked(int modId)

    // Scrolls just far enough to show row (of the model).
    function showRow(row) {
        wheel.stop()
        list.positionViewAtIndex(row, ListView.Contain)
    }

    ListView {
        id: list

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            leftMargin: 12
            right: scrollBar.left
            rightMargin: 6
        }
        topMargin: 2
        spacing: 6
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        // Also asks the model for the next page a screen before the end.
        cacheBuffer: height

        delegate: Rectangle {
            id: row

            required property int index
            required property int modId
            required property string name
            required property string author
            // Unix timestamp of the last update, in seconds.
            required property double updated
            required property string audio
            required property string thumbnail
            required property string category
            // GameBanana shows the mod behind a warning or hides it (see banana.Mod.IsNSFW).
            required property bool nsfw

            readonly property bool selected: modId === root.selectedModId

            width: ListView.view.width
            height: 44
            radius: 6
            color: selected ? Theme.highlightMed
                 : rowArea.containsMouse ? Qt.lighter(Theme.overlay, 1.15)
                 : Theme.overlay
            // Marks the sound that's open in the details panel, like the cards.
            border.width: selected ? 1 : 0
            border.color: Theme.gold

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            // Under everything else, so the player's controls get their own clicks.
            MouseArea {
                id: rowArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: root.modClicked(row.modId)
            }

            RoundedImage {
                id: thumbnailImage

                anchors {
                    left: parent.left
                    leftMargin: 6
                    verticalCenter: parent.verticalCenter
                }
                visible: !root.sounds
                width: 48
                height: 32
                radius: 4
                placeholderColor: Theme.highlightLow
                source: root.sounds ? "" : row.thumbnail
                blurred: row.nsfw
            }

            Label {
                anchors {
                    left: root.sounds ? parent.left : thumbnailImage.right
                    leftMargin: root.sounds ? 12 : 10
                    right: root.sounds ? player.left
                         : root.showCategory ? categoryLabel.left
                         : authorLabel.left
                    // For sounds: the player's play button is 6px in from its
                    // edge, so this ends the name 10px before the button.
                    rightMargin: root.sounds ? 4 : 10
                    verticalCenter: parent.verticalCenter
                }
                text: row.name
                color: Theme.text
                font.pixelSize: 13
                elide: Text.ElideRight
            }

            SoundPlayer {
                id: player

                visible: root.sounds
                anchors {
                    right: authorLabel.left
                    rightMargin: 4
                    verticalCenter: parent.verticalCenter
                }
                width: root.middleWidth
                color: "transparent"
                source: root.sounds ? row.audio : ""
                showVolume: false
                // The bubble would be cut off above the topmost rows.
                bubbleBelow: row.y - list.contentY < 32
            }

            // Where the player is for sounds, e.g. "Skins - Abrams".
            Label {
                id: categoryLabel

                anchors {
                    right: authorLabel.left
                    rightMargin: 12
                    verticalCenter: parent.verticalCenter
                }
                visible: root.showCategory
                width: Math.max(0, root.middleWidth)
                text: row.category
                color: Theme.subtle
                font.pixelSize: 12
                elide: Text.ElideRight
            }

            Label {
                id: authorLabel

                anchors {
                    right: root.showUpdated ? updatedRow.left : parent.right
                    rightMargin: 12
                    verticalCenter: parent.verticalCenter
                }
                width: 120
                text: row.author
                color: Theme.gold
                font.pixelSize: 12
                elide: Text.ElideRight
            }

            // When the sound was last updated, e.g. "8 hours ago".
            Row {
                id: updatedRow

                anchors {
                    right: parent.right
                    rightMargin: 12
                    verticalCenter: parent.verticalCenter
                }
                visible: root.showUpdated
                width: 116
                spacing: 4

                Image {
                    anchors.verticalCenter: parent.verticalCenter
                    width: 14
                    height: 14
                    source: "icons/calendar_ltr_24_regular.svg"
                    sourceSize: Qt.size(width, height)
                    opacity: 0.6
                }

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - 14 - parent.spacing
                    text: RelativeTime.format(row.updated, RelativeTime.now)
                    color: Theme.subtle
                    font.pixelSize: 12
                    elide: Text.ElideRight
                }
            }
        }

        footer: LoadMoreFooter {
            width: list.width
            loading: root.loading && list.count > 0
            error: list.count > 0 ? root.error : ""
            onRetry: root.retry()
        }

        ScrollBar.vertical: scrollBar

        // Emptied for a new listing, e.g. another category: start at the top.
        onCountChanged: {
            if (count === 0) {
                wheel.stop()
                positionViewAtBeginning()
            }
        }
    }

    SmoothWheel {
        id: wheel
        anchors.fill: list
        flickable: list
    }

    ScrollBar {
        id: scrollBar
        anchors {
            top: parent.top
            bottom: parent.bottom
            right: parent.right
        }
        policy: ScrollBar.AlwaysOn
        onPressedChanged: if (pressed) wheel.stop()
    }
}
