import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

Item {
    id: root

    // The category being browsed, as Breadcrumbs crumbs, from all mods down.
    property var crumbs: []

    signal crumbClicked(int depth)

    // Sounds have no preview images, so they're listed rather than shown as cards.
    readonly property bool sounds: app.section === "Sound"
    // Other mods can be shown either way, picked in the top bar.
    readonly property bool listed: sounds || topBar.viewIndex === 1
    readonly property var view: listed ? modList : modGrid
    readonly property int count: view.count
    readonly property int minimumWidth: topBar.minimumWidth + detailsPanel.Layout.preferredWidth

    // Whether the scroll position the app was left at has been restored, or
    // given up on. Until then, the position isn't saved: it would overwrite
    // the one being restored.
    property bool scrollRestored: false

    function toggleMod(modId) {
        details.id = details.id === modId ? 0 : modId
    }

    // Go looks for the mod that was at the top in the first few pages, then
    // bumps modsRestoreId. The view has to be shown to be scrolled.
    function restoreScroll() {
        if (scrollRestored || uiState.modsRestoreId === 0 || !visible)
            return

        scrollRestored = true
        view.restoreTop(uiState.modsRestoreRow, uiState.modsRestoreOffset)
    }

    function saveScroll() {
        if (!scrollRestored)
            return

        const top = view.topMod()

        uiState.modsTopMod = top.modId
        uiState.modsTopOffset = top.offset
    }

    Component.onCompleted: restoreScroll()

    onVisibleChanged: {
        // Nothing plays while the page isn't shown.
        if (!visible && soundPreview.state === "playing")
            soundPreview.state = "paused"

        restoreScroll()
    }

    readonly property int modsRestoreId: uiState.modsRestoreId
    onModsRestoreIdChanged: restoreScroll()

    Connections {
        target: root.view

        function onContentYChanged() {
            root.saveScroll()
        }
    }

    // Each pick of the best search result bumps pickId: show its row.
    readonly property int pickId: search.pickId
    onPickIdChanged: view.showRow(search.pickedRow)

    RowLayout {
        anchors.fill: parent
        spacing: 0

        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0

            TopBar {
                id: topBar

                Layout.fillWidth: true
                crumbs: root.crumbs
                searchPlaceholder: qsTr("Search Mods")
                // Only sounds have anything to play, and they're always listed.
                showVolume: root.sounds
                showViews: !root.sounds
                viewIndex: uiState.modsView
                onViewIndexChanged: uiState.modsView = viewIndex
                onCrumbClicked: (depth) => root.crumbClicked(depth)
                // Searches once the user pauses typing, rather than for every letter.
                onSearchEdited: searchDelay.restart()
                // Searches straight away, and opens the best match.
                onSearchAccepted: {
                    searchDelay.stop()
                    search.query = topBar.searchText
                    search.pick = true
                }

                Timer {
                    id: searchDelay

                    interval: 300
                    onTriggered: search.query = topBar.searchText
                }
            }

            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                ModGrid {
                    id: modGrid

                    anchors.fill: parent
                    visible: !root.listed
                    // Unset while hidden, so it doesn't build cards nobody sees.
                    model: root.listed ? null : mods
                    loading: app.loading
                    error: app.error
                    errorDetail: app.errorDetail
                    selectedModId: details.id
                    onRetry: retryAction.trigger()
                    onModClicked: (modId) => root.toggleMod(modId)
                }

                ModList {
                    id: modList

                    anchors.fill: parent
                    visible: root.listed
                    model: root.listed ? mods : null
                    sounds: root.sounds
                    loading: app.loading
                    error: app.error
                    errorDetail: app.errorDetail
                    selectedModId: details.id
                    onRetry: retryAction.trigger()
                    onModClicked: (modId) => root.toggleMod(modId)
                }

                // Initial load only; later pages show their progress at the end of the list.
                BusyIndicator {
                    anchors.centerIn: parent
                    running: app.loading && root.count === 0
                }

                Label {
                    anchors.centerIn: parent
                    visible: !app.loading && app.error === "" && root.count === 0 && search.query !== ""
                    text: qsTr("No Mods Found")
                    color: Theme.subtle
                }

                Column {
                    anchors.centerIn: parent
                    spacing: 12
                    visible: app.error !== "" && root.count === 0

                    ErrorLabel {
                        width: Math.min(implicitWidth, root.width - 32)
                        text: qsTr("Couldn’t load mods\n%1").arg(app.error)
                        detail: app.errorDetail
                        horizontalAlignment: Text.AlignHCenter
                    }

                    Button {
                        anchors.horizontalCenter: parent.horizontalCenter
                        text: qsTr("Retry")
                        onClicked: retryAction.trigger()
                    }
                }
            }
        }

        // Always visible, so opening a mod never changes the grid's width and
        // the cards stay where they are. With nothing open it shows a hint.
        ModDetailsPanel {
            id: detailsPanel

            Layout.preferredWidth: 275
            Layout.fillHeight: true

            open: details.id !== 0
            name: details.name
            author: details.author
            authorUrl: details.authorUrl
            image: details.image
            images: details.images
            audio: details.audio
            profileUrl: details.profileUrl
            description: details.description
            loading: details.loading
            error: details.error
            errorDetail: details.errorDetail
            // The sound's row still has its player.
            pauseOnClose: !root.sounds
            onImageClicked: (index) => viewer.show(details.images, index)
            onInstallClicked: installDialog.open()
        }
    }

    // Installs on the installed mods page, where it shows its progress.
    InstallDialog {
        id: installDialog

        name: details.name
        description: details.plainDescription
        files: details.files
        loading: details.loading
        error: details.error
        errorDetail: details.errorDetail
        onInstallRequested: (fileIds) => {
            details.install = fileIds
            close()
        }
    }

    ImageViewer {
        id: viewer

        onCopyImageRequested: (url) => clipboard.copyImage = url
        onCopyLinkRequested: (url) => clipboard.copyLink = url
    }

    // Each finished copy bumps messageId; show its message in the viewer.
    readonly property int copyMessageId: clipboard.messageId
    onCopyMessageIdChanged: viewer.notify(clipboard.message)

    Shortcut {
        sequence: "Escape"
        // The image viewer and the install dialog handle Escape themselves
        // while they're open.
        enabled: root.visible && details.id !== 0 && !viewer.visible && !installDialog.visible
        onActivated: details.id = 0
    }
}
