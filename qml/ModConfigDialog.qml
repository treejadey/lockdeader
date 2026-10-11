import QtQuick
import QtQuick.Controls.Fusion

AppDialog {
    id: root

    // As in installed.configuration: {key, name, loading, error, options},
    // where options are {id, name, detail, size, added, picked, installed}
    // objects.
    property var configuration: ({})
    readonly property var variants: configuration.options ?? []

    property var selectedIds: []
    readonly property bool onlyOne: selectedIds.length === 1
    // The picks differ from what the mod has.
    readonly property bool changed: {
        const enabled = variants.filter(v => v.picked).map(v => v.id)

        return enabled.length !== selectedIds.length || enabled.some(id => !isSelected(id))
    }

    signal changeRequested(string key, var ids)

    function isSelected(id) {
        return selectedIds.indexOf(id) >= 0
    }

    function toggle(id) {
        if (!isSelected(id))
            selectedIds = selectedIds.concat([id])
        else if (!onlyOne)
            selectedIds = selectedIds.filter(other => other !== id)
    }

    function reset() {
        selectedIds = variants.filter(v => v.picked).map(v => v.id)
    }

    // The list can change while it's open, e.g. once the mod's other files
    // on GameBanana have been looked up: picks already made are kept.
    onVariantsChanged: {
        if (changed)
            selectedIds = selectedIds.filter(id => variants.some(v => v.id === id))
        else
            reset()
    }

    onAboutToShow: reset()

    titlePrefix: configuration.name ?? ""
    title: "configuration"

    Rectangle {
        width: parent.width
        height: variantList.height + 2 * variantList.anchors.margins
        radius: 16
        color: "transparent"
        border.color: Theme.highlightMed

        ListView {
            id: variantList

            anchors {
                top: parent.top
                left: parent.left
                right: parent.right
                margins: 4
            }
            // Scrolls once there are more than about five.
            height: Math.min(contentHeight, 5.5 * 56)
            model: root.variants
            spacing: 4
            clip: true
            interactive: contentHeight > height
            boundsBehavior: Flickable.StopAtBounds
            ScrollBar.vertical: variantScrollBar

            delegate: AbstractButton {
                id: variant

                required property var modelData
                readonly property bool picked: root.isSelected(modelData.id)
                // The only one picked, which has to stay picked.
                readonly property bool locked: picked && root.onlyOne

                width: ListView.view.width - (variantList.interactive ? 12 : 0)
                implicitHeight: Math.max(52, variantText.implicitHeight + 16)
                leftPadding: 16
                rightPadding: 12
                hoverEnabled: true
                focusPolicy: Qt.TabFocus
                onClicked: root.toggle(modelData.id)

                HoverHandler {
                    cursorShape: variant.locked ? Qt.ArrowCursor : Qt.PointingHandCursor
                }

                background: Rectangle {
                    radius: 12
                    // Clear while locked, so its antialiased edge doesn't
                    // show around the disabled look's.
                    color: variant.locked ? "transparent"
                         : variant.picked ? Theme.highlightMed
                         : variant.hovered ? Qt.lighter(Theme.overlay, 1.25)
                         : "transparent"
                    border.width: variant.visualFocus ? 1 : 0
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
                        shown: variant.locked
                    }
                }

                contentItem: Item {
                    Item {
                        id: icon

                        anchors.verticalCenter: parent.verticalCenter
                        width: 22
                        height: 22

                        Image {
                            anchors.fill: parent
                            source: "icons/folder_zip_24_regular.svg"
                            sourceSize: Qt.size(width, height)
                            opacity: variant.picked ? 0 : 0.8

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
                            opacity: variant.picked ? 1 : 0

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
                        id: variantText

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
                            text: variant.modelData.name
                            color: Theme.text
                            elide: Text.ElideMiddle
                        }

                        // What it is, e.g. "Flag only", or for a VPK, where
                        // it is in the archive.
                        Label {
                            id: detailLabel

                            width: parent.width
                            visible: text !== ""
                            text: variant.modelData.detail
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

                    Row {
                        id: addedRow

                        anchors {
                            right: sizeLabel.left
                            rightMargin: 12
                            verticalCenter: parent.verticalCenter
                        }
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
                            text: RelativeTime.format(variant.modelData.added, RelativeTime.now)
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
                        text: variant.modelData.size
                        color: variant.picked ? Theme.text : Theme.subtle

                        Behavior on color {
                            ColorAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }

                    // Shown on the picked variants, and on the one the
                    // pointer is over, as clicking it would pick it.
                    TickBox {
                        id: check

                        anchors {
                            right: parent.right
                            verticalCenter: parent.verticalCenter
                        }
                        checked: variant.picked
                        enabled: !variant.locked
                        labelHovered: variant.hovered
                        // The row takes the focus instead.
                        focusPolicy: Qt.NoFocus
                        opacity: variant.picked || variant.hovered || variant.visualFocus ? 1 : 0
                        onClicked: root.toggle(variant.modelData.id)

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
            id: variantWheel

            anchors.fill: variantList
            flickable: variantList
            enabled: variantList.interactive
        }

        // Fusion draws the handle in palette.mid, which is the dialog's own
        // colour, so it gets lighter ones here.
        ScrollBar {
            id: variantScrollBar

            anchors {
                top: variantList.top
                bottom: variantList.bottom
                right: variantList.right
            }
            visible: variantList.interactive
            policy: ScrollBar.AlwaysOn
            palette.mid: Theme.highlightMed
            palette.midlight: Theme.highlightHigh
            onPressedChanged: if (pressed) variantWheel.stop()
        }
    }

    // While the mod's files on GameBanana are being looked up, only its
    // installed options are listed.
    Row {
        x: 2
        width: parent.width - 2
        spacing: 8
        visible: root.configuration.loading === true || (root.configuration.error ?? "") !== ""

        BusyIndicator {
            anchors.verticalCenter: parent.verticalCenter
            width: 18
            height: 18
            running: root.configuration.loading === true
            visible: running
        }

        Label {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - (root.configuration.loading === true ? 26 : 0)
            text: root.configuration.loading === true ? "Looking for other versions on GameBanana…"
                : "Couldn't look for other versions: " + root.configuration.error
            color: root.configuration.loading === true ? Theme.subtle : Theme.love
            elide: Text.ElideRight
        }
    }

    // Hidden with opacity, so it keeps its room and the dialog doesn't change
    // size.
    Label {
        x: 2
        width: parent.width - 2
        text: "At least one variant must be selected."
        color: Theme.subtle
        font.weight: Font.DemiBold
        wrapMode: Text.Wrap
        opacity: root.onlyOne ? 1 : 0

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    buttons: ActionButton {
        width: Math.max(implicitWidth, 218)
        implicitHeight: 44
        text: "Change configuration"
        baseColor: Theme.foam
        textColor: Theme.base
        enabled: root.changed
        onClicked: {
            root.changeRequested(root.configuration.key, root.selectedIds)
            root.close()
        }
    }
}
