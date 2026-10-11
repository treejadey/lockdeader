import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    readonly property bool listed: topBar.viewIndex === 1
    readonly property var view: listed ? list : grid
    // As on the mods page: until the scroll position is restored, it isn't
    // saved.
    property bool scrollRestored: false

    signal browseModsRequested()
    // Asks to take the mods from other tools over (see TakeOverDialog).
    signal takeOverRequested()

    function restoreScroll() {
        if (scrollRestored || !visible)
            return

        scrollRestored = true
        view.restoreTop(uiState.installedTop, uiState.installedTopOffset)
    }

    function saveScroll() {
        if (!scrollRestored)
            return

        const top = view.topItem()

        uiState.installedTop = Math.max(0, top.index)
        uiState.installedTopOffset = top.offset
    }

    // Once laid out: the mods are already loaded from the database.
    Component.onCompleted: Qt.callLater(restoreScroll)

    Connections {
        target: root.view

        function onContentYChanged() {
            root.saveScroll()
        }
    }

    // Looks up which files the mods have on GameBanana, so their menus know
    // whether there are other versions to pick from; and looks through the
    // game's addons folders again, in case another tool changed them.
    onVisibleChanged: {
        if (visible) {
            installed.refresh = true
            otherMods.scan = true
        }

        restoreScroll()
    }

    readonly property int count: listed ? list.count : grid.count

    function setEnabled(key, enabled) {
        installed.setEnabled = { key: key, enabled: enabled }
    }

    function showMenu(key, status, variants) {
        menu.key = key
        menu.status = status
        menu.variants = variants
        menu.popup()
    }

    // Mods from other tools, under the installed ones: a heading and the
    // same cards or rows, which only show their state. It takes no room
    // while there are none.
    Component {
        id: otherSectionGrid

        Column {
            width: GridView.view ? GridView.view.width : 0
            visible: otherMods.count > 0
            height: visible ? implicitHeight : 0

            OtherModsHeading {}

            InstalledGrid {
                width: parent.width
                embedded: true
                readOnly: true
                model: otherModsModel
                onMenuRequested: otherMenu.popup()
            }
        }
    }

    Component {
        id: otherSectionList

        Column {
            width: ListView.view ? ListView.view.width : 0
            visible: otherMods.count > 0
            height: visible ? implicitHeight : 0

            OtherModsHeading {}

            InstalledList {
                width: parent.width
                embedded: true
                readOnly: true
                model: otherModsModel
                onMenuRequested: otherMenu.popup()
            }
        }
    }

    component OtherModsHeading: Label {
        width: parent.width
        topPadding: 18
        bottomPadding: 4
        leftPadding: 12
        text: qsTr("From Other Tools")
        color: Theme.subtle
        font.weight: Font.DemiBold
    }

    TopBar {
        id: topBar

        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
        }
        crumbs: [{ name: qsTr("Installed Mods"), depth: 0 }]
        searchPlaceholder: qsTr("Search Installed Mods")
        showViews: true
        viewIndex: uiState.installedView
        onViewIndexChanged: uiState.installedView = viewIndex
        onSearchEdited: installed.query = topBar.searchText
        onSearchAccepted: installed.query = topBar.searchText
    }

    // Why the mods aren't reaching the game, until they do.
    InfoBar {
        id: applyBar

        anchors {
            top: topBar.bottom
            left: parent.left
            right: parent.right
        }
        shown: installed.applyError !== ""
        severity: "error"
        text: qsTr("Couldn’t put the mods into the game. ") + installed.applyError
        detail: installed.applyErrorDetail
        action: qsTr("Retry")
        onActionTriggered: installed.applyNow = true
    }

    // Mods other tools put in the game can be taken over (see TakeOverDialog).
    InfoBar {
        id: otherBar

        anchors {
            top: applyBar.bottom
            left: parent.left
            right: parent.right
        }
        shown: otherMods.count > 0
        severity: "information"
        text: otherMods.count === 1
            ? qsTr("A mod from another tool was found in the game’s addons folder. Lockdeader can take it over.")
            : qsTr("%1 mods from other tools were found in the game’s addons folder. Lockdeader can take them over.").arg(otherMods.count)
        action: qsTr("Take Over…")
        onActionTriggered: root.takeOverRequested()
    }

    Item {
        anchors {
            top: otherBar.bottom
            left: parent.left
            right: parent.right
            bottom: parent.bottom
        }

        InstalledGrid {
            id: grid

            anchors.fill: parent
            visible: !root.listed
            model: root.listed ? null : installedMods
            footer: otherSectionGrid
            onEnabledToggled: (key, enabled) => root.setEnabled(key, enabled)
            onMenuRequested: (key, status, variants) => root.showMenu(key, status, variants)
        }

        InstalledList {
            id: list

            anchors.fill: parent
            visible: root.listed
            model: root.listed ? installedMods : null
            footer: otherSectionList
            onEnabledToggled: (key, enabled) => root.setEnabled(key, enabled)
            onMenuRequested: (key, status, variants) => root.showMenu(key, status, variants)
        }

        Column {
            anchors.centerIn: parent
            spacing: 6
            visible: installed.total === 0 && otherMods.count === 0

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("No Mods Installed Yet")
                color: Theme.text
                font.pixelSize: 18
                font.weight: Font.DemiBold
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Mods you install will show up here")
                color: Theme.subtle
            }

            // Spacing on top of the column's, to set the action apart.
            Item {
                width: 1
                height: 6
            }

            ActionButton {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Browse Mods")
                baseColor: Theme.accent
                textColor: Theme.textOnAccent
                onClicked: root.browseModsRequested()
            }
        }

        Label {
            anchors.centerIn: parent
            visible: installed.total > 0 && root.count === 0
            text: qsTr("No Installed Mods Found")
            color: Theme.subtle
        }
    }

    PopupMenu {
        id: menu

        property string key
        property string status
        property int variants

        PopupMenuItem {
            text: qsTr("Retry")
            icon.source: "icons/arrow_clockwise_24_regular.svg"
            shown: menu.status === "failed"
            onTriggered: installed.retry = menu.key
        }

        PopupMenuItem {
            text: menu.status === "failed" ? qsTr("Dismiss") : qsTr("Cancel Install")
            icon.source: "icons/dismiss_24_regular.svg"
            shown: menu.status !== ""
            onTriggered: installed.cancel = menu.key
        }

        PopupMenuItem {
            text: qsTr("Change Configuration…")
            icon.source: "icons/options_24_regular.svg"
            shown: menu.status === "" && menu.variants > 1
            onTriggered: {
                installed.configure = menu.key
                configDialog.open()
            }
        }

        PopupMenuItem {
            text: qsTr("Uninstall")
            icon.source: "icons/delete_24_regular.svg"
            shown: menu.status === ""
            onTriggered: installed.uninstall = menu.key
        }
    }

    // Other tools' mods can only be looked at where they are.
    PopupMenu {
        id: otherMenu

        PopupMenuItem {
            text: qsTr("Open Folder")
            icon.source: "icons/folder_24_regular.svg"
            onTriggered: Qt.openUrlExternally(otherMods.folderUrl)
        }
    }

    ModConfigDialog {
        id: configDialog

        configuration: installed.configuration
        onChangeRequested: (key, ids) => installed.setConfiguration = { key: key, ids: ids }
    }
}
