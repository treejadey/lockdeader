import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// Browse GameBanana's mods in a grid, with the selected mod's details in a
// panel on the right. Clicking the open mod again, or pressing Escape, closes it.
Item {
    id: root

    RowLayout {
        anchors.fill: parent
        spacing: 0

        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            ModGrid {
                id: modGrid

                anchors.fill: parent
                model: mods
                loading: app.loading
                error: app.error
                selectedModId: details.id
                onRetry: retryAction.trigger()
                onModClicked: (modId) => {
                    details.id = details.id === modId ? 0 : modId
                }
            }

            // Initial load only; later pages show their progress at the end of the grid.
            BusyIndicator {
                anchors.centerIn: parent
                running: app.loading && modGrid.count === 0
            }

            Column {
                anchors.centerIn: parent
                spacing: 12
                visible: app.error !== "" && modGrid.count === 0

                Label {
                    width: Math.min(implicitWidth, root.width - 32)
                    text: "Couldn't load mods:\n" + app.error
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.Wrap
                }

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Retry"
                    onClicked: retryAction.trigger()
                }
            }
        }

        // Always visible, so opening a mod never changes the grid's width and
        // the cards stay where they are. With nothing open it shows a hint.
        ModDetailsPanel {
            Layout.preferredWidth: 275
            Layout.fillHeight: true

            open: details.id !== 0
            name: details.name
            author: details.author
            image: details.image
            images: details.images
            profileUrl: details.profileUrl
            description: details.description
            loading: details.loading
            error: details.error
            onImageClicked: (index) => viewer.show(details.images, index)
        }
    }

    ImageViewer {
        id: viewer
    }

    Shortcut {
        sequence: "Escape"
        // The image viewer handles Escape itself while it's open.
        enabled: root.visible && details.id !== 0 && !viewer.visible
        onActivated: details.id = 0
    }
}
