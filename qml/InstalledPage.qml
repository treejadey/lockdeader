import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    readonly property bool listed: topBar.viewIndex === 1

    signal browseModsRequested()

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
        crumbs: [{ name: qsTr("Installed Mods"), depth: 0 }]
        searchPlaceholder: qsTr("Search Installed Mods")
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
                baseColor: Theme.foam
                textColor: Theme.base
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

    ModConfigDialog {
        id: configDialog

        configuration: installed.configuration
        onChangeRequested: (key, ids) => installed.setConfiguration = { key: key, ids: ids }
    }
}
