import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// The app's main navigation. currentIndex is the page to show:
// 0 = view mods, 1 = installed mods, 2 = settings.
Rectangle {
    id: root

    property int currentIndex: 0

    color: Theme.surface

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 4
        spacing: 4

        Label {
            Layout.leftMargin: 6
            Layout.topMargin: 4
            Layout.bottomMargin: 2
            text: "Mods"
            color: Theme.subtle
        }

        SidebarItem {
            Layout.fillWidth: true
            text: "View mods"
            icon.source: "icons/apps_add_in_24_regular.svg"
            active: root.currentIndex === 0
            onClicked: root.currentIndex = 0
        }

        SidebarItem {
            Layout.fillWidth: true
            text: "Installed mods"
            icon.source: "icons/apps_24_regular.svg"
            active: root.currentIndex === 1
            onClicked: root.currentIndex = 1
        }

        Item {
            Layout.fillHeight: true
        }

        SidebarItem {
            Layout.fillWidth: true
            text: "Settings"
            icon.source: "icons/settings_24_regular.svg"
            active: root.currentIndex === 2
            onClicked: root.currentIndex = 2
        }
    }
}
