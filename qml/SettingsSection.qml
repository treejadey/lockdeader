import QtQuick
import QtQuick.Controls.Fusion

Column {
    id: root

    property string title
    // The settings page's search; "" for everything.
    property string query
    readonly property bool hasMatches: matchCount > 0

    default property alias rows: rowColumn.data

    property int matchCount: 0

    readonly property string needle: query.trim().toLowerCase()

    function filter() {
        // When the title matches, every row shows.
        const titleMatches = needle === "" || title.toLowerCase().includes(needle)
        let count = 0

        for (const row of rowColumn.children) {
            // Not counted by reading visible back: that's false while the
            // section itself is hidden.
            const shown = titleMatches || row.matches(needle)
            row.visible = shown

            if (shown)
                count++
        }

        matchCount = count
    }

    onNeedleChanged: filter()
    Component.onCompleted: filter()

    visible: hasMatches
    spacing: 12

    Item {
        width: parent.width
        height: heading.implicitHeight

        Label {
            id: heading

            // Lines up with the names of the settings in the rows.
            x: 16
            text: root.title
            color: Theme.subtle
            font.pixelSize: 24
            font.weight: Font.Light
        }

        Rectangle {
            anchors {
                left: heading.right
                leftMargin: 16
                right: parent.right
                rightMargin: 4
                verticalCenter: heading.verticalCenter
            }
            height: 2
            radius: 1
            color: Theme.highlightMed
        }
    }

    Column {
        id: rowColumn

        width: parent.width
        spacing: 4
    }
}
