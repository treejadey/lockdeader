import QtQuick
import QtQuick.Controls.Fusion

import "ScrollPosition.js" as ScrollPosition

Item {
    id: root

    property alias model: grid.model
    property alias footer: grid.footer
    readonly property alias contentY: grid.contentY
    readonly property alias count: grid.count
    // Mods that can only be looked at, e.g. other tools': the checkbox
    // shows their state but does nothing.
    property bool readOnly: false
    // Laid out inside something else that scrolls, e.g. a footer: as tall
    // as its contents, without a scrollbar.
    property bool embedded: false
    property int cardWidth: 240
    property int cardHeight: Math.round(cardWidth * 9 / 16)
    // Under the picture, for the name and date.
    property int textHeight: 48
    // The edges of the grid get at least half of it.
    property int cardSpacing: 12
    // In pixels per mouse wheel notch.
    property real wheelStep: 100

    signal enabledToggled(string key, bool enabled)
    signal menuRequested(string key, string status, int variants)

    function topItem() {
        return ScrollPosition.top(grid)
    }

    function restoreTop(index, offset) {
        wheel.stop()
        ScrollPosition.restore(grid, index, offset)
    }

    GridView {
        id: grid

        // Same as GridView's own column count, which ignores the margins.
        readonly property int columns: Math.max(1, Math.floor(width / cellWidth))
        // Room under each card for an install's progress. Every cell has it, so
        // cards don't move as installs start and finish.
        readonly property int statusHeight: 28

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            right: root.embedded ? parent.right : scrollBar.left
        }
        topMargin: root.cardSpacing / 2
        clip: !root.embedded
        interactive: !root.embedded
        boundsBehavior: Flickable.StopAtBounds

        cellWidth: root.cardWidth + root.cardSpacing
        cellHeight: root.cardHeight + root.textHeight + root.cardSpacing + statusHeight

        delegate: Item {
            id: cell

            required property string key
            required property int modId
            required property string name
            required property double updated
            required property string thumbnail
            required property bool nsfw
            required property bool modEnabled
            required property string status
            required property real progress
            required property string pace
            required property string error
            required property string errorDetail
            required property int variants

            readonly property bool installing: status !== ""
            // Where the tile is in the cell: centred, then shifted with the
            // block of columns (see ModGrid).
            readonly property real tileX: (width - root.cardWidth) / 2 + card.horizontalOffset

            width: grid.cellWidth
            height: grid.cellHeight

            ModCard {
                id: card

                width: grid.cellWidth
                height: root.cardHeight + root.textHeight + root.cardSpacing
                modId: cell.modId
                name: cell.name
                updated: cell.updated
                // Sounds have no picture on GameBanana, so they get the app's.
                thumbnail: cell.thumbnail !== "" || !cell.key.startsWith("Sound/") ? cell.thumbnail
                    : Qt.resolvedUrl("icons/bitmap/sound_icon.png")
                nsfw: cell.nsfw
                tileWidth: root.cardWidth
                tileHeight: root.cardHeight
                textHeight: root.textHeight
                horizontalOffset: Math.floor((grid.width - grid.columns * grid.cellWidth) / 2)
                dimmed: cell.installing || !cell.modEnabled
                onClicked: if (!cell.installing && !root.readOnly) root.enabledToggled(cell.key, !cell.modEnabled)
                onMenuRequested: root.menuRequested(cell.key, cell.status, cell.variants)

                TickBox {
                    id: check

                    anchors {
                        right: parent.right
                        bottom: parent.bottom
                        margins: 8
                    }
                    checked: cell.modEnabled
                    enabled: !cell.installing && !root.readOnly
                    labelHovered: card.hovered && !root.readOnly
                    onClicked: root.enabledToggled(cell.key, !cell.modEnabled)
                }
            }

            InstallStatus {
                x: cell.tileX
                y: root.cardSpacing / 2 + root.cardHeight + root.textHeight
                width: root.cardWidth
                visible: cell.installing
                status: cell.status
                progress: cell.progress
                pace: cell.pace
                error: cell.error
                errorDetail: cell.errorDetail
            }
        }

        ScrollBar.vertical: root.embedded ? null : scrollBar
    }

    // Embedded, the grid is as tall as its rows, and the parent scrolls.
    implicitHeight: root.embedded ? grid.contentHeight + grid.topMargin : 0

    SmoothWheel {
        id: wheel

        anchors.fill: grid
        enabled: !root.embedded
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
        visible: !root.embedded
        policy: ScrollBar.AlwaysOn
        onPressedChanged: if (pressed) wheel.stop()
    }
}
