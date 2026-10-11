import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    readonly property bool listed: topBar.viewIndex === 1

    // Looks up which files the mods have on GameBanana, so their menus know
    // whether there are other versions to pick from.
    onVisibleChanged: if (visible) installed.refresh = true

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

    TopBar {
        id: topBar

        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
        }
        crumbs: [{ name: "Installed mods", depth: 0 }]
        searchPlaceholder: "Search installed mods.."
        showViews: true
        onSearchEdited: installed.query = topBar.searchText
        onSearchAccepted: installed.query = topBar.searchText
    }

    Item {
        anchors {
            top: topBar.bottom
            left: parent.left
            right: parent.right
            bottom: parent.bottom
        }

        InstalledGrid {
            id: grid

            anchors.fill: parent
            visible: !root.listed
            model: root.listed ? null : installedMods
            onEnabledToggled: (key, enabled) => root.setEnabled(key, enabled)
            onMenuRequested: (key, status, variants) => root.showMenu(key, status, variants)
        }

        InstalledList {
            id: list

            anchors.fill: parent
            visible: root.listed
            model: root.listed ? installedMods : null
            onEnabledToggled: (key, enabled) => root.setEnabled(key, enabled)
            onMenuRequested: (key, status, variants) => root.showMenu(key, status, variants)
        }

        Column {
            anchors.centerIn: parent
            spacing: 6
            visible: installed.total === 0

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "No mods installed yet"
                color: Theme.text
                font.pixelSize: 18
                font.weight: Font.DemiBold
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Mods you install will show up here."
                color: Theme.subtle
            }
        }

        Label {
            anchors.centerIn: parent
            visible: installed.total > 0 && root.count === 0
            text: "No installed mods found"
            color: Theme.subtle
        }
    }

    PopupMenu {
        id: menu

        property string key
        property string status
        property int variants

        // A hidden item would still take up its height in the menu.
        PopupMenuItem {
            text: "Retry"
            icon.source: "icons/arrow_clockwise_24_regular.svg"
            visible: menu.status === "failed"
            height: visible ? implicitHeight : 0
            onTriggered: installed.retry = menu.key
        }

        PopupMenuItem {
            text: menu.status === "failed" ? "Dismiss" : "Cancel install"
            icon.source: "icons/dismiss_24_regular.svg"
            visible: menu.status !== ""
            height: visible ? implicitHeight : 0
            onTriggered: installed.cancel = menu.key
        }

        PopupMenuItem {
            text: "Change configuration…"
            icon.source: "icons/options_24_regular.svg"
            visible: menu.status === "" && menu.variants > 1
            height: visible ? implicitHeight : 0
            onTriggered: {
                installed.configure = menu.key
                configDialog.open()
            }
        }

        PopupMenuItem {
            text: "Uninstall"
            icon.source: "icons/delete_24_regular.svg"
            visible: menu.status === ""
            height: visible ? implicitHeight : 0
            onTriggered: installed.uninstall = menu.key
        }
    }

    ModConfigDialog {
        id: configDialog

        configuration: installed.configuration
        onChangeRequested: (key, ids) => installed.setConfiguration = { key: key, ids: ids }
    }
}
