import QtQuick
import QtQuick.Controls.Fusion

// A vertically scrolling grid of fixed-size mod cards, centred horizontally.
Item {
    id: root

    property alias model: grid.model
    readonly property alias count: grid.count
    // Shown at the bottom of the grid while the next page is on its way.
    property bool loading: false
    property string error: ""
    property int cardSize: 150
    // Gap between cards; the edges of the grid get at least half of it.
    property int cardSpacing: 12
    // How far one mouse wheel notch scrolls, in pixels.
    property real wheelStep: 100
    // The card with this mod id is drawn as selected; 0 for none.
    property int selectedModId: 0

    // Emitted when the user asks to retry after a failed page load.
    signal retry()
    signal modClicked(int modId)

    GridView {
        id: grid

        // Same as GridView's own column count, which ignores the margins.
        readonly property int columns: Math.max(1, Math.floor(width / cellWidth))

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            right: scrollBar.left
        }
        topMargin: root.cardSpacing / 2
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        // Build cards (and start their image downloads) a screen ahead of the
        // viewport. This also makes the view ask the model for the next page
        // about a screen before reaching the end, so scrolling never stalls.
        cacheBuffer: height

        // Each cell is a card plus half a gap on every side. Cell size never
        // depends on the content, so nothing moves while thumbnails load.
        cellWidth: root.cardSize + root.cardSpacing
        cellHeight: cellWidth

        delegate: ModCard {
            width: grid.cellWidth
            height: grid.cellHeight
            tileSize: root.cardSize
            // GridView lays columns out from the left; shift every card by half
            // the unused width so the block of columns is centred.
            horizontalOffset: Math.floor((grid.width - grid.columns * grid.cellWidth) / 2)
            selected: modId === root.selectedModId
            onClicked: root.modClicked(modId)
        }

        // Fixed height whether or not anything is showing, so the content
        // doesn't jump when loading starts or stops.
        footer: Item {
            width: grid.width
            height: 56

            BusyIndicator {
                anchors.centerIn: parent
                running: root.loading && grid.count > 0
            }

            Row {
                anchors.centerIn: parent
                spacing: 12
                visible: root.error !== "" && grid.count > 0

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    width: Math.min(implicitWidth, grid.width - retryButton.width - parent.spacing - 32)
                    text: "Couldn't load more mods: " + root.error
                    elide: Text.ElideRight
                }

                Button {
                    id: retryButton
                    text: "Retry"
                    onClicked: root.retry()
                }
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
