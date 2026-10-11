import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// Shown instead of the app when the settings file fails to load and there are
// backups of it. The answer goes to the restore bridge (restoredialog.go).
AppWindow {
    id: window

    property int selectedSlot: restore.backups.length > 0 ? restore.backups[0].slot : 0
    property bool answered: false

    function answer(choice) {
        answered = true
        restore.choice = choice
    }

    // A fixed size, all bound to this rather than to height, which changes
    // as the window is resized to fit.
    readonly property int fittedHeight: content.implicitHeight + 2 * content.anchors.margins

    width: 540
    height: fittedHeight
    minimumWidth: 540
    maximumWidth: 540
    minimumHeight: fittedHeight
    maximumHeight: fittedHeight
    flags: Qt.Dialog
    visible: true
    title: qsTr("Restore Settings — Lockdeader")

    onClosing: if (!answered) answer(-1)

    ColumnLayout {
        id: content

        // Not filling the window: its height comes from this.
        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
            margins: 24
        }
        spacing: 16
        // Keeps text from wobbling on fractionally scaled displays.
        transform: PixelNudge {}

        Label {
            Layout.fillWidth: true
            text: qsTr("Your settings couldn’t be loaded")
            color: Theme.text
            font.pixelSize: 18
            font.weight: Font.DemiBold
            wrapMode: Text.Wrap
        }

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: errorText.implicitHeight + 20
            radius: 6
            color: Theme.overlay

            Label {
                id: errorText

                anchors {
                    left: parent.left
                    right: parent.right
                    verticalCenter: parent.verticalCenter
                    margins: 12
                }
                text: restore.error
                color: Theme.subtle
                font.family: app.monospaceFont
                font.pixelSize: 12
                wrapMode: Text.WrapAnywhere
            }
        }

        Label {
            Layout.fillWidth: true
            text: qsTr("There are backups of your settings from earlier sessions. Restore one, "
                + "or carry on without your settings for now: you’ll get the defaults, and "
                + "nothing you change will be saved, so the file stays as it is for you to fix.")
            color: Theme.text
            wrapMode: Text.Wrap
        }

        Column {
            Layout.fillWidth: true
            spacing: 4

            Repeater {
                model: restore.backups

                AbstractButton {
                    id: backup

                    required property var modelData
                    readonly property bool selected: modelData.slot === window.selectedSlot

                    width: parent.width
                    implicitHeight: 40
                    leftPadding: 12
                    rightPadding: 12
                    hoverEnabled: true
                    focusPolicy: Qt.TabFocus
                    onPressed: window.selectedSlot = modelData.slot
                    onDoubleClicked: window.answer(modelData.slot)
                    Keys.onReturnPressed: window.answer(modelData.slot)

                    HoverHandler {
                        cursorShape: Qt.PointingHandCursor
                    }

                    background: Rectangle {
                        radius: 6
                        color: backup.selected ? Theme.highlightMed
                             : backup.hovered ? Theme.lift(Theme.overlay, 0.25)
                             : Theme.overlay
                        border.width: backup.visualFocus ? 1 : 0
                        border.color: Theme.iris

                        Behavior on color {
                            ColorAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }

                    contentItem: RowLayout {
                        spacing: 12

                        Label {
                            Layout.fillWidth: true
                            text: new Date(backup.modelData.modified * 1000)
                                .toLocaleString(Qt.locale(), Locale.ShortFormat)
                            color: Theme.text
                            elide: Text.ElideRight
                        }

                        Label {
                            text: RelativeTime.format(backup.modelData.modified, RelativeTime.now)
                            color: Theme.subtle
                        }
                    }
                }
            }
        }

        Label {
            Layout.fillWidth: true
            visible: restore.damaged
            text: qsTr("Some backups are damaged, so they can’t be restored.")
            color: Theme.gold
            wrapMode: Text.Wrap
        }

        Label {
            Layout.fillWidth: true
            text: qsTr("Restoring keeps your current file as Settings.toml.broken.")
            color: Theme.subtle
            font.pixelSize: 12
            wrapMode: Text.Wrap
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.topMargin: 8
            spacing: 8

            ActionButton {
                text: qsTr("Open Folder")
                onClicked: Qt.openUrlExternally(restore.settingsDirUrl)
            }

            Item {
                Layout.fillWidth: true
            }

            ActionButton {
                text: qsTr("Quit")
                onClicked: window.answer(-1)
            }

            ActionButton {
                text: qsTr("Use Defaults")
                onClicked: window.answer(0)
            }

            ActionButton {
                text: qsTr("Restore Backup")
                baseColor: Theme.highlightMed
                enabled: window.selectedSlot > 0
                onClicked: window.answer(window.selectedSlot)
            }
        }
    }
}
