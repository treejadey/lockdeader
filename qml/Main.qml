import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

AppWindow {
    id: window

    width: 1300
    height: 730
    // The mods page is the widest; every page has room at this width.
    minimumWidth: sidebar.Layout.preferredWidth + modsPage.minimumWidth
    minimumHeight: 560
    visible: true
    title: "lockdeader"

    RowLayout {
        anchors.fill: parent
        spacing: 0
        // Keeps text from wobbling on fractionally scaled displays.
        transform: PixelNudge {}

        Sidebar {
            id: sidebar
            settingsSections: settingsPage.sectionList
            currentSettingsSection: settingsPage.currentSection
            onSettingsSectionClicked: (index) => settingsPage.goToSection(index)
            Layout.preferredWidth: 240
            Layout.fillHeight: true
        }

        StackLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: sidebar.currentIndex

            ModsPage {
                id: modsPage

                crumbs: sidebar.categoryPath
                onCrumbClicked: (depth) => sidebar.goToCategory(depth)
            }

            InstalledPage {}

            SettingsPage {
                id: settingsPage
            }
        }
    }

    // Says what just happened, e.g. that a mod has started downloading, near
    // the top of the window. It's in the overlay, over any dialog, so it isn't
    // hidden by one closing as it appears.
    Item {
        parent: Overlay.overlay
        anchors.fill: parent
        z: 1000
        // Popups don't get the nudge on the window's content.
        transform: PixelNudge {}

        Toast {
            id: toast

            anchors.horizontalCenter: parent.horizontalCenter
            y: 16
            maximumWidth: Math.min(420, parent.width - 32)
        }
    }

    // Each message from the installer bumps messageId; show it.
    readonly property int installMessageId: installed.messageId
    onInstallMessageIdChanged: toast.show(installed.message)
}
