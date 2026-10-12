import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

Rectangle {
    id: root

    // 0 = view mods, 1 = installed mods, 2 = settings.
    property int currentIndex: 0
    // As Breadcrumbs crumbs.
    readonly property alias categoryPath: categoryNav.path
    property alias settingsSections: settingsNav.sections
    property alias currentSettingsSection: settingsNav.currentIndex

    signal settingsSectionClicked(int index)

    // Takes the depth of a crumb in categoryPath.
    function goToCategory(depth) {
        categoryNav.goTo(depth)
    }

    color: Theme.surface

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 4
        spacing: 4

        Label {
            Layout.leftMargin: 6
            Layout.topMargin: 4
            Layout.bottomMargin: 2

            text: qsTr("Mods")
            color: Theme.subtle
        }

        SidebarItem {
            Layout.fillWidth: true

            text: qsTr("View Mods")
            icon.source: "icons/apps_add_in_24_regular.svg"
            active: root.currentIndex === 0
            onClicked: {
                // Clicking it again goes back to all mods.
                if (root.currentIndex === 0)
                    categoryNav.reset()

                root.currentIndex = 0
            }
        }

        SidebarItem {
            Layout.fillWidth: true

            text: qsTr("Installed Mods")
            icon.source: "icons/apps_24_regular.svg"
            badge: installed.installing
            active: root.currentIndex === 1
            onClicked: root.currentIndex = 1
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.leftMargin: 6
            Layout.rightMargin: 6
            Layout.topMargin: 4
            Layout.bottomMargin: 4

            implicitHeight: 1
            color: Theme.highlightMed
            // Over the options for the page, for pages that have them.
            visible: root.currentIndex !== 1
        }

        // The options for each page, in page order.
        StackLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true

            currentIndex: root.currentIndex

            CategoryNav {
                id: categoryNav
            }

            // Installed mods has none.
            Item {}

            SettingsNav {
                id: settingsNav

                onSectionClicked: (index) => root.settingsSectionClicked(index)
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.topMargin: 4

            implicitHeight: 38

            ActionButton {
                anchors.fill: parent
                text: qsTr("Play With Mods")
                baseColor: Theme.accent
                textColor: Theme.textOnAccent
                sheen: true
                // Disabled rather than hidden, so it's clear there's a way to
                // play with mods once some are installed and the game is
                // known.
                enabled: installed.installedCount > 0 && settings.gameFolder !== ""
                         && !launcher.gameRunning
                onClicked: launcher.play = "mods"
            }

            // Pressing it without a game folder still gets an answer: Go
            // says to choose one, with a way to the settings. A disabled
            // button takes no clicks, so this sits over it.
            MouseArea {
                anchors.fill: parent
                enabled: settings.gameFolder === "" && !launcher.gameRunning
                onClicked: launcher.play = "mods"
            }

            // A disabled button doesn't see the pointer, so its parent does.
            HoverHandler {
                id: playHover
            }

            ToolTip {
                visible: playHover.hovered && launcher.gameRunning
                delay: 500
                text: qsTr("Deadlock is running")
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.bottomMargin: 4

            implicitHeight: 38

            ActionButton {
                anchors.fill: parent
                text: qsTr("Play Without Mods")
                enabled: !launcher.gameRunning
                onClicked: launcher.play = "vanilla"
            }

            HoverHandler {
                id: playVanillaHover
            }

            ToolTip {
                visible: playVanillaHover.hovered && launcher.gameRunning
                delay: 500
                text: qsTr("Deadlock is running")
            }
        }

        SidebarItem {
            Layout.fillWidth: true

            text: qsTr("Settings")
            icon.source: "icons/settings_24_regular.svg"
            active: root.currentIndex === 2
            onClicked: root.currentIndex = 2
        }
    }
}
