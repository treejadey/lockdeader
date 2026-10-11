import QtQuick
import QtQuick.Controls.Fusion

// As the list narrows, the progress bar shrinks first, then when it was
// updated goes.
Item {
    id: root

    property alias model: list.model
    readonly property alias count: list.count

    // The columns, from the right: the checkbox, when it was updated, the
    // author, and the progress of an install. Each column's width includes
    // the gap before it.
    readonly property int checkWidth: 22 + 12
    readonly property int updatedWidth: 116 + 12
    readonly property int authorWidth: 120 + 12
    readonly property int statusMinWidth: 120
    readonly property int statusMaxWidth: 240
    readonly property int nameMinWidth: 140
    readonly property bool showUpdated: list.width - 12 - nameMinWidth - statusMinWidth
        - authorWidth - updatedWidth - checkWidth >= 0
    readonly property real statusWidth: Math.max(statusMinWidth, Math.min(statusMaxWidth,
        list.width - 12 - nameMinWidth - authorWidth - (showUpdated ? updatedWidth : 0) - checkWidth - 22))

    signal enabledToggled(string key, bool enabled)
    // Emitted on a right click.
    signal menuRequested(string key, string status, int variants)

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

        delegate: Rectangle {
            id: row

            required property string key
            required property string name
            required property string author
            // Unix timestamp of the last update, in seconds.
            required property double updated
            required property bool modEnabled
            required property string status
            required property real progress
            required property string pace
            required property string error
            required property int variants

            readonly property bool installing: status !== ""

            width: ListView.view.width
            height: 44
            radius: 6
            color: rowArea.containsMouse && !installing ? Qt.lighter(Theme.overlay, 1.15) : Theme.overlay

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            // Under everything else, so the checkbox gets its own clicks.
            MouseArea {
                id: rowArea

                anchors.fill: parent
                hoverEnabled: true
                acceptedButtons: Qt.LeftButton | Qt.RightButton
                cursorShape: row.installing ? Qt.ArrowCursor : Qt.PointingHandCursor
                onClicked: (mouse) => {
                    if (mouse.button === Qt.RightButton)
                        root.menuRequested(row.key, row.status, row.variants)
                    else if (!row.installing)
                        root.enabledToggled(row.key, !row.modEnabled)
                }
            }

            Label {
                anchors {
                    left: parent.left
                    leftMargin: 12
                    right: row.installing ? installStatus.left : authorLabel.left
                    rightMargin: row.installing ? 22 : 12
                    verticalCenter: parent.verticalCenter
                }
                text: row.name
                color: Theme.text
                font.pixelSize: 13
                elide: Text.ElideRight
            }

            InstallStatus {
                id: installStatus

                anchors {
                    right: authorLabel.left
                    rightMargin: 12
                    verticalCenter: parent.verticalCenter
                }
                width: root.statusWidth
                visible: row.installing
                labelBeside: true
                trackColor: Theme.surface
                status: row.status
                progress: row.progress
                pace: row.pace
                error: row.error
            }

            Label {
                id: authorLabel

                anchors {
                    right: root.showUpdated ? updatedRow.left : check.left
                    rightMargin: 12
                    verticalCenter: parent.verticalCenter
                }
                width: 120
                text: row.author
                color: Theme.gold
                font.pixelSize: 12
                elide: Text.ElideRight
            }

            Row {
                id: updatedRow

                anchors {
                    right: check.left
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

            TickBox {
                id: check

                anchors {
                    right: parent.right
                    rightMargin: 11
                    verticalCenter: parent.verticalCenter
                }
                checked: row.modEnabled
                enabled: !row.installing
                labelHovered: rowArea.containsMouse
                onClicked: root.enabledToggled(row.key, !row.modEnabled)
            }
        }

        ScrollBar.vertical: scrollBar
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
