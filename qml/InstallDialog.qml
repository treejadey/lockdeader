import QtQuick
import QtQuick.Controls.Fusion

// Motion: opening a modal is an expressive moment, so the dialog rises from
// low down towards the middle of the window as it fades in, with Carbon's
// expressive entrance, over slow-01, as it moves a long way. The backdrop dims
// over Carbon's background dimming duration. Closing is a quick productive
// exit that drops it back slightly.
Popup {
    id: root

    property string name
    // Plain text.
    property string description
    // {id, name, size, added, description} objects; size is formatted, e.g.
    // "450 KB", and added is when the file was uploaded, in Unix seconds.
    property var files: []
    property bool loading: false
    property string error: ""
    property string errorDetail: ""

    property var selectedIds: []
    readonly property bool single: files.length === 1

    signal installRequested(var fileIds)

    function isSelected(id) {
        return selectedIds.indexOf(id) >= 0
    }

    function toggle(id) {
        if (single)
            return

        selectedIds = isSelected(id)
            ? selectedIds.filter(other => other !== id)
            : selectedIds.concat([id])
    }

    // The first file is the one most mods want installed.
    function pickDefault() {
        selectedIds = files.length > 0 ? [files[0].id] : []
    }

    onFilesChanged: pickDefault()
    onAboutToShow: pickDefault()

    parent: Overlay.overlay
    x: 0
    y: 0
    width: parent ? parent.width : 0
    height: parent ? parent.height : 0
    padding: 0

    modal: true
    dim: false
    focus: true
    closePolicy: Popup.CloseOnEscape
    background: null

    enter: Transition {
        ParallelAnimation {
            NumberAnimation {
                target: backdrop
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.slow02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }

            NumberAnimation {
                target: card
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.slow01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }

            // From most of the way down to the bottom of the window.
            NumberAnimation {
                target: cardShift
                property: "y"
                from: (root.height - card.y) * 0.6
                to: 0
                duration: Motion.slow01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.expressiveEntrance
            }
        }
    }

    exit: Transition {
        ParallelAnimation {
            NumberAnimation {
                targets: [backdrop, card]
                property: "opacity"
                to: 0
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }

            NumberAnimation {
                target: cardShift
                property: "y"
                to: 24
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveExit
            }
        }
    }

    Rectangle {
        id: backdrop

        anchors.fill: parent
        color: Theme.backdrop

        // A MouseArea rather than a TapHandler: pointer handlers see every
        // press under them, even ones the dialog above has already taken.
        MouseArea {
            anchors.fill: parent
            onClicked: root.close()
        }
    }

    Rectangle {
        id: card

        anchors.centerIn: parent
        width: Math.min(600, parent.width - 48)
        height: content.implicitHeight + 2 * content.anchors.margins
        radius: 16
        color: Theme.overlay
        border.color: Theme.highlightMed
        // Keeps text from wobbling on fractionally scaled displays. The dialog
        // is a popup, so the nudge on the main window's content doesn't reach it.
        transform: [
            Translate {
                id: cardShift
            },
            PixelNudge {}
        ]

        // Clicks on the dialog stay on it, rather than closing it.
        MouseArea {
            anchors.fill: parent
        }

        Column {
            id: content

            anchors {
                top: parent.top
                left: parent.left
                right: parent.right
                margins: 28
            }
            spacing: 16

            Label {
                width: parent.width - closeButton.width - 8
                text: root.name
                color: Theme.text
                font.pixelSize: 22
                wrapMode: Text.Wrap
                maximumLineCount: 2
                elide: Text.ElideRight
            }

            // The start of the description, fading out over its last lines.
            Item {
                id: descriptionBox

                readonly property real maxHeight: 6 * descriptionMetrics.height

                width: parent.width
                height: Math.min(descriptionText.implicitHeight, maxHeight)
                visible: root.description !== ""
                clip: true

                FontMetrics {
                    id: descriptionMetrics

                    font: descriptionText.font
                }

                Label {
                    id: descriptionText

                    width: parent.width
                    text: root.description
                    color: Theme.subtle
                    wrapMode: Text.Wrap
                }

                Rectangle {
                    anchors {
                        left: parent.left
                        right: parent.right
                        bottom: parent.bottom
                    }
                    height: 3 * descriptionMetrics.height
                    visible: descriptionText.implicitHeight > descriptionBox.maxHeight
                    gradient: Gradient {
                        GradientStop { position: 0; color: Qt.rgba(card.color.r, card.color.g, card.color.b, 0) }
                        GradientStop { position: 1; color: card.color }
                    }
                }
            }

            FilePickerList {
                files: root.files
                detailKey: "description"
                selectedIds: root.selectedIds
                // A mod's only file is always installed.
                keepOne: root.single
                visibleRows: 4.5
                // Room for the spinner or the message while there are no files.
                emptyHeight: 56
                onToggled: (id) => root.toggle(id)

                BusyIndicator {
                    anchors.centerIn: parent
                    width: 32
                    height: 32
                    running: root.loading && root.files.length === 0
                    visible: running
                }

                ErrorLabel {
                    anchors {
                        left: parent.left
                        right: parent.right
                        verticalCenter: parent.verticalCenter
                        margins: 16
                    }
                    visible: !root.loading && root.files.length === 0
                    text: root.error !== "" ? qsTr("Couldn’t load the files: %1").arg(root.error)
                                            : qsTr("This mod has no files to install.")
                    detail: root.errorDetail
                    color: root.error !== "" ? Theme.love : Theme.subtle
                    horizontalAlignment: Text.AlignHCenter
                }
            }

            // Some room above the button, as in Carbon's modals.
            Item {
                width: parent.width
                height: installButton.implicitHeight + 24

                ActionButton {
                    id: installButton

                    anchors {
                        right: parent.right
                        bottom: parent.bottom
                    }
                    width: Math.max(implicitWidth, 180)
                    implicitHeight: 42
                    text: root.single ? qsTr("Install") : qsTr("Install Selected")
                    baseColor: Theme.accent
                    textColor: Theme.textOnAccent
                    enabled: root.selectedIds.length > 0
                    onClicked: root.installRequested(root.selectedIds)
                }
            }
        }

        AbstractButton {
            id: closeButton

            anchors {
                top: parent.top
                right: parent.right
                margins: 14
            }
            width: 32
            height: 32
            padding: 6
            hoverEnabled: true
            focusPolicy: Qt.TabFocus
            onClicked: root.close()

            HoverHandler {
                cursorShape: Qt.PointingHandCursor
            }

            ToolTip.visible: hovered
            ToolTip.delay: 500
            ToolTip.text: qsTr("Close (Esc)")

            background: Rectangle {
                radius: 6
                color: closeButton.hovered ? Theme.highlightMed : "transparent"
                border.width: closeButton.visualFocus ? 1 : 0
                border.color: Theme.iris

                Behavior on color {
                    ColorAnimation {
                        duration: Motion.fast01
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveStandard
                    }
                }
            }

            contentItem: TintedIcon {
                source: "icons/dismiss_24_regular.svg"
                opacity: closeButton.hovered ? 1 : 0.7
            }
        }
    }
}
