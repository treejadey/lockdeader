import QtQuick
import QtQuick.Controls.Fusion

// In the style of Carbon's dropdown: a field like SearchBox showing the
// option picked, with the list of options opening straight under it.
// Not named ComboBox: Qt Quick Controls has a type of that name, which would
// win over this file. Like TickBox, picking emits picked() without changing
// currentId, so whatever it's bound to decides.
AbstractButton {
    id: root

    // {id, name} objects.
    property var options: []
    property var currentId

    readonly property var current: options.find(o => o.id === currentId)

    signal picked(var id)

    // The options in the list are laid out the same, with the tick in the
    // chevron's place, so they fit the same width.
    implicitWidth: leftPadding + sizer.implicitWidth + spacing + chevron.width + rightPadding
    implicitHeight: 36
    leftPadding: 16
    // Less than on the left: the chevron's glyph has room around it in its
    // icon, which would make the gap look wider than the text's.
    rightPadding: 12
    spacing: 8
    hoverEnabled: true
    focusPolicy: Qt.TabFocus
    onClicked: list.visible ? list.close() : list.open()

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }

    // As wide as the widest option, so the field doesn't change size with
    // the pick.
    Column {
        id: sizer

        visible: false

        Repeater {
            model: root.options

            Label {
                required property var modelData

                text: modelData.name
                font: label.font
            }
        }
    }

    background: Rectangle {
        topLeftRadius: 4
        topRightRadius: 4
        color: root.hovered && !list.visible ? Theme.lift(Theme.overlay, 0.15) : Theme.overlay

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        // The line along the bottom, as SearchBox's.
        Rectangle {
            anchors {
                left: parent.left
                right: parent.right
                bottom: parent.bottom
            }
            height: list.visible || root.visualFocus ? 2 : 1
            color: list.visible || root.visualFocus ? Theme.foam : Theme.highlightHigh

            Behavior on height {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
    }

    contentItem: Item {
        Label {
            id: label

            anchors.verticalCenter: parent.verticalCenter
            text: root.current?.name ?? ""
            color: Theme.text
        }

        TintedIcon {
            id: chevron

            anchors {
                right: parent.right
                verticalCenter: parent.verticalCenter
            }
            width: 16
            height: 16
            source: list.visible ? "icons/chevron_down_24_filled.svg" : "icons/chevron_down_24_regular.svg"
            // Points up while the list is open, as Carbon's does.
            rotation: list.visible ? 180 : 0

            Behavior on rotation {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
    }

    Popup {
        id: list

        // As in PopupMenu: drawn in the window, not as a native popup.
        popupType: Popup.Item
        y: root.height
        width: root.width
        padding: 0

        onAboutToShow: {
            view.currentIndex = Math.max(0, root.options.findIndex(o => o.id === root.currentId))
            view.forceActiveFocus()
        }

        function pick(index) {
            root.picked(root.options[index].id)
            close()
        }

        enter: Transition {
            NumberAnimation {
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveEntrance
            }
        }

        exit: Transition {
            NumberAnimation {
                property: "opacity"
                from: 1
                to: 0
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }
        }

        background: Rectangle {
            color: Theme.overlay
            border.color: Theme.highlightMed
        }

        contentItem: ListView {
            id: view

            implicitHeight: contentHeight
            model: root.options
            interactive: false
            keyNavigationEnabled: true
            clip: true

            Keys.onReturnPressed: list.pick(currentIndex)
            Keys.onEnterPressed: list.pick(currentIndex)
            Keys.onSpacePressed: list.pick(currentIndex)

            delegate: AbstractButton {
                id: option

                required property var modelData
                required property int index
                readonly property bool isCurrent: modelData.id === root.currentId

                width: ListView.view.width
                implicitHeight: 40
                leftPadding: root.leftPadding
                rightPadding: root.rightPadding
                hoverEnabled: true
                focusPolicy: Qt.NoFocus
                onHoveredChanged: if (hovered) view.currentIndex = index
                onClicked: list.pick(index)

                background: Rectangle {
                    color: option.isCurrent ? Theme.highlightMed
                         : option.ListView.isCurrentItem ? Theme.lift(Theme.overlay, 0.15)
                         : "transparent"

                    // Between options, inset as Carbon's, and not beside the
                    // highlighted one.
                    Rectangle {
                        anchors {
                            left: parent.left
                            right: parent.right
                            top: parent.top
                            leftMargin: 16
                            rightMargin: 16
                        }
                        height: 1
                        color: Theme.highlightMed
                        visible: option.index > 0 && !option.ListView.isCurrentItem && !option.isCurrent
                                 && view.currentIndex !== option.index - 1
                                 && root.options[option.index - 1].id !== root.currentId
                    }
                }

                contentItem: Item {
                    Label {
                        anchors {
                            left: parent.left
                            right: tick.left
                            rightMargin: root.spacing
                            verticalCenter: parent.verticalCenter
                        }
                        text: option.modelData.name
                        color: option.isCurrent || option.ListView.isCurrentItem ? Theme.text : Theme.subtle
                        elide: Text.ElideRight
                    }

                    TintedIcon {
                        id: tick

                        anchors {
                            right: parent.right
                            verticalCenter: parent.verticalCenter
                        }
                        width: 16
                        height: 16
                        source: "icons/checkmark_24_filled.svg"
                        visible: option.isCurrent
                    }
                }
            }
        }
    }
}
