import QtQuick
import QtQuick.Controls.Fusion

// Installed mods as a grid of cards, like ModGrid's: each with a checkbox in
// its bottom-right corner that turns it on or off, as does clicking the card.
// A mod that's installing is dimmed, its checkbox greyed out, and its
// progress is shown under it.
Item {
    id: root

    property alias model: grid.model
    readonly property alias count: grid.count
    property int cardSize: 150
    // Gap between cards; the edges of the grid get at least half of it.
    property int cardSpacing: 12
    // How far one mouse wheel notch scrolls, in pixels.
    property real wheelStep: 100

    // Emitted when the user turns a mod on or off.
    signal enabledToggled(string key, bool enabled)
    // Emitted on a right click on a mod, with its status and how many
    // variants it has.
    signal menuRequested(string key, string status, int variants)

    GridView {
        id: grid

        // Same as GridView's own column count, which ignores the margins.
        readonly property int columns: Math.max(1, Math.floor(width / cellWidth))
        // Room under each card for the progress of an install. Every cell has
        // it, so cards don't move as installs start and finish.
        readonly property int statusHeight: 28

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            right: scrollBar.left
        }
        topMargin: root.cardSpacing / 2
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        cellWidth: root.cardSize + root.cardSpacing
        cellHeight: cellWidth + statusHeight

        delegate: Item {
            id: cell

            required property string key
            required property int modId
            required property string name
            required property string author
            required property double updated
            required property string thumbnail
            required property bool nsfw
            required property bool modEnabled
            required property string status
            required property real progress
            required property string pace
            required property string error
            required property int variants

            readonly property bool installing: status !== ""
            // Where the tile is in the cell: centred, then shifted with the
            // block of columns (see ModGrid).
            readonly property real tileX: (width - root.cardSize) / 2 + card.horizontalOffset

            width: grid.cellWidth
            height: grid.cellHeight

            ModCard {
                id: card

                width: grid.cellWidth
                height: grid.cellWidth
                modId: cell.modId
                name: cell.name
                author: cell.author
                updated: cell.updated
                thumbnail: cell.thumbnail
                nsfw: cell.nsfw
                tileSize: root.cardSize
                horizontalOffset: Math.floor((grid.width - grid.columns * grid.cellWidth) / 2)
                // Like the checkbox, the author waits for the install to finish.
                showAuthor: !cell.installing
                dimmed: cell.installing
                cornerRightMargin: check.width + 6
                onClicked: if (!cell.installing) root.enabledToggled(cell.key, !cell.modEnabled)
                onMenuRequested: root.menuRequested(cell.key, cell.status, cell.variants)

                TickBox {
                    id: check

                    anchors {
                        right: parent.right
                        bottom: parent.bottom
                        margins: 8
                    }
                    checked: cell.modEnabled
                    enabled: !cell.installing
                    labelHovered: card.hovered
                    onClicked: root.enabledToggled(cell.key, !cell.modEnabled)
                }
            }

            InstallStatus {
                x: cell.tileX
                y: (grid.cellWidth - root.cardSize) / 2 + root.cardSize + 8
                width: root.cardSize
                visible: cell.installing
                status: cell.status
                progress: cell.progress
                pace: cell.pace
                error: cell.error
            }
        }

        ScrollBar.vertical: scrollBar
    }

    SmoothWheel {
        id: wheel
        anchors.fill: grid
        flickable: grid
        step: root.wheelStep
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
