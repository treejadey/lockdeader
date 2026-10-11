import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// The app's main navigation. currentIndex is the page to show:
// 0 = view mods, 1 = installed mods, 2 = settings. Under the page buttons
// are options for the page being shown: categories for view mods, and the
// sections of the settings page.
Rectangle {
    id: root

    property int currentIndex: 0
    // The category being browsed, as Breadcrumbs crumbs.
    readonly property alias categoryPath: categoryNav.path
    // The settings page's sections, as SettingsNav's sections, and the one
    // being read.
    property alias settingsSections: settingsNav.sections
    property alias currentSettingsSection: settingsNav.currentIndex

    // Emitted when a section of the settings page is clicked, to go to it.
    signal settingsSectionClicked(int index)

    // Goes back up to a category in categoryPath, by its crumb's depth, so
    // the category list shows it picked.
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
            text: "Mods"
            color: Theme.subtle
        }

        SidebarItem {
            Layout.fillWidth: true
            text: "View mods"
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
            text: "Installed mods"
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

            Item {}

            SettingsNav {
                id: settingsNav
                onSectionClicked: (index) => root.settingsSectionClicked(index)
            }
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
