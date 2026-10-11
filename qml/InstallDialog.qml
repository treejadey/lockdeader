import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Effects

// Asks which of a mod's files to install: its name and the start of its
// description, then its files, any number of which can be picked (the first
// is, to begin with). A mod with just one file has nothing to pick: that file
// is always picked, can't be un-picked, and has no checkbox. The window behind is dimmed, as for the image viewer.
// Escape, the close button or clicking outside closes it.
//
// Motion: opening a modal is an expressive moment, so the dialog rises from
// low down towards the middle of the window as it fades in, with Carbon's
// expressive entrance, over slow-01, as it moves a long way. The backdrop dims
// over Carbon's background dimming duration. Closing is a quick productive
// exit that drops it back slightly.
Popup {
    id: root

    property string name
    // Plain text; only its start shows, fading out.
    property string description
    // {id, name, size, added, description} objects; size is formatted, e.g.
    // "450 KB", and added is when the file was uploaded, in Unix seconds.
    property var files: []
    // The files are still on their way, or failed to load.
    property bool loading: false
    property string error: ""

    // The ids of the files picked.
    property var selectedIds: []
    // There's only one file, so there's no choosing.
    readonly property bool single: files.length === 1

    // Emitted by the Install selected button, with the ids of the files picked.
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

    // Picks the first file, as the one most mods want installed.
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
        color: Qt.rgba(Theme.base.r, Theme.base.g, Theme.base.b, 0.92)

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

            // The files, in a rounded box, with the same space around them on
            // every side.
            Rectangle {
                width: parent.width
                // Room for the spinner or the message while there are no files.
                height: (root.files.length > 0 ? fileList.height : 56) + 2 * fileList.anchors.margins
                radius: 14
                color: "transparent"
                border.color: Theme.highlightMed

                ListView {
                    id: fileList

                    anchors {
                        top: parent.top
                        left: parent.left
                        right: parent.right
                        margins: 6
                    }
                    // Scrolls once there are more than about four.
                    height: Math.min(contentHeight, 4.5 * 54)
                    model: root.files
                    spacing: 4
                    clip: true
                    interactive: contentHeight > height
                    boundsBehavior: Flickable.StopAtBounds
                    ScrollBar.vertical: fileScrollBar

                    delegate: AbstractButton {
                        id: file

                        required property var modelData
                        readonly property bool picked: root.isSelected(modelData.id)

                        width: ListView.view.width - (fileList.interactive ? 12 : 0)
                        implicitHeight: Math.max(50, fileText.implicitHeight + 16)
                        leftPadding: 12
                        rightPadding: 12
                        hoverEnabled: !root.single
                        focusPolicy: root.single ? Qt.NoFocus : Qt.TabFocus
                        onClicked: root.toggle(modelData.id)

                        HoverHandler {
                            enabled: !root.single
                            cursorShape: Qt.PointingHandCursor
                        }

                        background: Rectangle {
                            radius: 10
                            color: file.picked ? (file.hovered ? Qt.lighter(Theme.foam, 1.08) : Theme.foam)
                                 : file.hovered ? Qt.lighter(Theme.overlay, 1.25)
                                 : "transparent"
                            // Only shown when focused with the keyboard.
                            border.width: file.visualFocus ? 1 : 0
                            border.color: Theme.iris

                            Behavior on color {
                                ColorAnimation {
                                    duration: Motion.fast01
                                    easing.type: Easing.BezierSpline
                                    easing.bezierCurve: Motion.productiveStandard
                                }
                            }
                        }

                        // Light text on the dialog, dark on a picked file's foam.
                        contentItem: Item {
                            id: fileContent

                            readonly property color textColor: file.picked ? Theme.base : Theme.text

                            // The outline (_regular) and filled icons are
                            // cross-faded. The filled ones are dark, for foam.
                            Item {
                                id: fileIcon

                                anchors.verticalCenter: parent.verticalCenter
                                width: 22
                                height: 22

                                Image {
                                    anchors.fill: parent
                                    source: "icons/folder_zip_24_regular.svg"
                                    sourceSize: Qt.size(width, height)
                                    opacity: file.picked ? 0 : 0.8

                                    Behavior on opacity {
                                        NumberAnimation {
                                            duration: Motion.fast01
                                            easing.type: Easing.BezierSpline
                                            easing.bezierCurve: Motion.productiveStandard
                                        }
                                    }
                                }

                                Image {
                                    anchors.fill: parent
                                    source: "icons/folder_zip_24_filled.svg"
                                    sourceSize: Qt.size(width, height)
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
                                    left: fileIcon.right
                                    leftMargin: 12
                                    right: addedRow.left
                                    rightMargin: 12
                                    verticalCenter: parent.verticalCenter
                                }
                                spacing: 1

                                Label {
                                    width: parent.width
                                    text: file.modelData.name
                                    color: fileContent.textColor
                                    elide: Text.ElideMiddle
                                }

                                // e.g. which variant of the mod the file is. Hovering
                                // it shows all of it, when it's cut short.
                                Label {
                                    id: fileDescription

                                    width: parent.width
                                    visible: text !== ""
                                    text: file.modelData.description
                                    color: fileContent.textColor
                                    opacity: 0.7
                                    font.pixelSize: 12
                                    elide: Text.ElideRight

                                    HoverHandler {
                                        id: descriptionHover
                                    }

                                    ToolTip {
                                        visible: descriptionHover.hovered && fileDescription.truncated
                                        delay: 500
                                        text: file.modelData.description
                                        // Long descriptions wrap rather than spanning the window.
                                        width: Math.min(implicitWidth, 360)
                                    }
                                }
                            }

                            // When the file was uploaded, e.g. "3 days ago", just
                            // before the size. Files can't be changed once
                            // uploaded, only replaced.
                            Row {
                                id: addedRow

                                anchors {
                                    right: sizeLabel.left
                                    rightMargin: 10
                                    verticalCenter: parent.verticalCenter
                                }
                                spacing: 4
                                opacity: 0.8

                                Image {
                                    id: calendarIcon

                                    anchors.verticalCenter: parent.verticalCenter
                                    width: 14
                                    height: 14
                                    source: "icons/calendar_ltr_24_regular.svg"
                                    sourceSize: Qt.size(width, height)
                                    // The icon is light; dark on a picked file's foam.
                                    layer.enabled: file.picked
                                    layer.effect: MultiEffect {
                                        colorization: 1
                                        colorizationColor: Theme.base
                                        brightness: -1
                                    }
                                }

                                Label {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: RelativeTime.format(file.modelData.added, RelativeTime.now)
                                    color: fileContent.textColor
                                    font.pixelSize: 12
                                }
                            }

                            Label {
                                id: sizeLabel

                                anchors {
                                    right: check.visible ? check.left : parent.right
                                    rightMargin: check.visible ? 12 : 0
                                    verticalCenter: parent.verticalCenter
                                }
                                text: file.modelData.size
                                color: fileContent.textColor
                                opacity: 0.8
                            }

                            Item {
                                id: check

                                anchors {
                                    right: parent.right
                                    verticalCenter: parent.verticalCenter
                                }
                                width: 22
                                height: 22
                                visible: !root.single

                                Image {
                                    anchors.fill: parent
                                    source: "icons/checkbox_unchecked_24_regular.svg"
                                    sourceSize: Qt.size(width, height)
                                    opacity: file.picked ? 0 : 0.5

                                    Behavior on opacity {
                                        NumberAnimation {
                                            duration: Motion.fast01
                                            easing.type: Easing.BezierSpline
                                            easing.bezierCurve: Motion.productiveStandard
                                        }
                                    }
                                }

                                Image {
                                    anchors.fill: parent
                                    source: "icons/checkbox_checked_24_filled.svg"
                                    sourceSize: Qt.size(width, height)
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
                        }
                    }
                }

                SmoothWheel {
                    id: fileWheel
                    anchors.fill: fileList
                    flickable: fileList
                    enabled: fileList.interactive
                }

                // Fusion draws the handle in palette.mid, which is the
                // dialog's own colour, so it gets lighter ones here.
                ScrollBar {
                    id: fileScrollBar

                    anchors {
                        top: fileList.top
                        bottom: fileList.bottom
                        right: fileList.right
                    }
                    visible: fileList.interactive
                    policy: ScrollBar.AlwaysOn
                    palette.mid: Theme.highlightMed
                    palette.midlight: Theme.highlightHigh
                    onPressedChanged: if (pressed) fileWheel.stop()
                }

                // While there are no files to show.
                BusyIndicator {
                    anchors.centerIn: parent
                    width: 32
                    height: 32
                    running: root.loading && root.files.length === 0
                    visible: running
                }

                Label {
                    anchors {
                        left: parent.left
                        right: parent.right
                        verticalCenter: parent.verticalCenter
                        margins: 16
                    }
                    visible: !root.loading && root.files.length === 0
                    text: root.error !== "" ? "Couldn't load the files: " + root.error
                                            : "This mod has no files to install."
                    color: root.error !== "" ? Theme.love : Theme.subtle
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.Wrap
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
                    text: root.single ? "Install" : "Install selected"
                    baseColor: Theme.foam
                    textColor: Theme.base
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
            ToolTip.text: "Close (Esc)"

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

            contentItem: Image {
                source: "icons/dismiss_24_regular.svg"
                sourceSize: Qt.size(width, height)
                opacity: closeButton.hovered ? 1 : 0.7
            }
        }
    }
}
