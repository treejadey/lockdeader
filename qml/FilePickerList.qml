import QtQuick
import QtQuick.Controls.Fusion

// A framed list of a mod's files to pick from, as in the install and
// configuration dialogs. The dialog keeps the picks: clicking a file emits
// toggled(), and the dialog updates selectedIds.
Rectangle {
    id: root

    // {id, name, size, added} objects, plus a line about each under detailKey.
    // size is formatted, e.g. "450 KB"; added is in Unix seconds.
    property var files: []
    property string detailKey: "detail"
    property var selectedIds: []
    // Whether the only picked file has to stay picked. It's shown disabled,
    // since clicking it does nothing.
    property bool keepOne: false
    // Rows shown before the list scrolls.
    property real visibleRows: 5.5
    // Room for something else while there are no files, e.g. a spinner.
    property real emptyHeight: 0

    readonly property bool onlyOne: selectedIds.length === 1

    signal toggled(var id)

    function isSelected(id) {
        return selectedIds.indexOf(id) >= 0
    }

    width: parent.width
    height: (files.length > 0 ? list.height : emptyHeight) + 2 * list.anchors.margins
    radius: 16
    color: "transparent"
    border.color: Theme.highlightMed

    ListView {
        id: list

        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
            margins: 4
        }
        height: Math.min(contentHeight, root.visibleRows * 56)
        model: root.files
        spacing: 4
        clip: true
        interactive: contentHeight > height
        boundsBehavior: Flickable.StopAtBounds
        ScrollBar.vertical: scrollBar

        delegate: AbstractButton {
            id: file

            required property var modelData
            readonly property bool picked: root.isSelected(modelData.id)
            readonly property bool locked: picked && root.keepOne && root.onlyOne

            width: ListView.view.width - (list.interactive ? 12 : 0)
            implicitHeight: Math.max(52, fileText.implicitHeight + 16)
            leftPadding: 16
            rightPadding: 12
            hoverEnabled: true
            focusPolicy: Qt.TabFocus
            onClicked: if (!locked) root.toggled(modelData.id)

            HoverHandler {
                cursorShape: file.locked ? Qt.ArrowCursor : Qt.PointingHandCursor
            }

            background: Rectangle {
                radius: 12
                // Clear while locked, so its antialiased edge doesn't show
                // around the disabled look's.
                color: file.locked ? "transparent"
                     : file.picked ? Theme.highlightMed
                     : file.hovered ? Theme.lift(Theme.overlay, 0.25)
                     : "transparent"
                border.width: file.visualFocus ? 1 : 0
                border.color: Theme.iris

                Behavior on color {
                    ColorAnimation {
                        duration: Motion.fast01
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }

                DisabledFill {
                    anchors.fill: parent
                    radius: parent.radius
                    shown: file.locked
                }
            }

            contentItem: Item {
                Item {
                    id: icon

                    anchors.verticalCenter: parent.verticalCenter
                    width: 22
                    height: 22

                    TintedIcon {
                        anchors.fill: parent
                        source: "icons/folder_zip_24_regular.svg"
                        opacity: file.picked ? 0 : 0.8

                        Behavior on opacity {
                            NumberAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }

                    TintedIcon {
                        anchors.fill: parent
                        source: "icons/folder_zip_24_filled.svg"
                        color: Theme.text
                        opacity: file.picked ? 1 : 0

                        Behavior on opacity {
                            NumberAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }
                }

                Column {
                    id: fileText

                    anchors {
                        left: icon.right
                        leftMargin: 12
                        right: addedRow.left
                        rightMargin: 12
                        verticalCenter: parent.verticalCenter
                    }
                    spacing: 1

                    Label {
                        width: parent.width
                        text: file.modelData.name
                        color: Theme.text
                        elide: Text.ElideMiddle
                    }

                    // e.g. which variant of the mod it is, or for a VPK, where
                    // it is in the archive.
                    Label {
                        id: detailLabel

                        width: parent.width
                        visible: text !== ""
                        text: file.modelData[root.detailKey] ?? ""
                        color: Theme.subtle
                        font.pixelSize: 12
                        elide: Text.ElideRight

                        HoverHandler {
                            id: detailHover
                        }

                        ToolTip {
                            visible: detailHover.hovered && detailLabel.truncated
                            delay: 500
                            text: detailLabel.text
                            width: Math.min(implicitWidth, 360)
                        }
                    }
                }

                // When the file was uploaded. Files can't be changed once
                // uploaded, only replaced.
                Row {
                    id: addedRow

                    anchors {
                        right: sizeLabel.left
                        rightMargin: 12
                        verticalCenter: parent.verticalCenter
                    }
                    spacing: 4

                    TintedIcon {
                        anchors.verticalCenter: parent.verticalCenter
                        width: 14
                        height: 14
                        source: "icons/calendar_ltr_24_regular.svg"
                        opacity: 0.6
                    }

                    Label {
                        anchors.verticalCenter: parent.verticalCenter
                        text: RelativeTime.format(file.modelData.added, RelativeTime.now)
                        color: Theme.subtle
                        font.pixelSize: 12
                    }
                }

                Label {
                    id: sizeLabel

                    anchors {
                        right: check.left
                        rightMargin: 12
                        verticalCenter: parent.verticalCenter
                    }
                    text: file.modelData.size
                    color: file.picked ? Theme.text : Theme.subtle

                    Behavior on color {
                        ColorAnimation {
                            duration: Motion.fast01
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.productiveStandard
                        }
                    }
                }

                // Shown on the picked files, and on the one the pointer is
                // over, as clicking it would pick it.
                TickBox {
                    id: check

                    anchors {
                        right: parent.right
                        verticalCenter: parent.verticalCenter
                    }
                    checked: file.picked
                    enabled: !file.locked
                    labelHovered: file.hovered
                    // The row takes the focus instead.
                    focusPolicy: Qt.NoFocus
                    opacity: file.picked || file.hovered || file.visualFocus ? 1 : 0
                    onClicked: root.toggled(file.modelData.id)

                    Behavior on opacity {
                        NumberAnimation {
                            duration: Motion.fast02
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.productiveStandard
                        }
                    }
                }
            }
        }
    }

    SmoothWheel {
        id: wheel

        anchors.fill: list
        flickable: list
        enabled: list.interactive
    }

    // Fusion draws the handle in palette.mid, which is the dialog's own
    // colour, so it gets lighter ones here.
    ScrollBar {
        id: scrollBar

        anchors {
            top: list.top
            bottom: list.bottom
            right: list.right
        }
        visible: list.interactive
        policy: ScrollBar.AlwaysOn
        palette.mid: Theme.highlightMed
        palette.midlight: Theme.highlightHigh
        onPressedChanged: if (pressed) wheel.stop()
    }
}
