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
    title: "Lockdeader"

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

    // In the overlay, over any dialog, so a dialog closing as a toast appears
    // doesn't hide it.
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

    // Another copy of the app was started, which asks this one to show its
    // window, closed or not.
    readonly property int showRequests: app.showRequests

    onShowRequestsChanged: {
        if (visibility === Window.Minimized)
            showNormal()
        else
            show()

        raise()
        requestActivate()
    }

    // Each message from the installer bumps messageId, so the same message
    // twice still shows.
    readonly property int installMessageId: installed.messageId
    // The installer message the toast shows, so its Undo or its going away can
    // be passed back.
    property int toastMessageId: 0

    // Showing a message dismisses the one before, which still has the old id.
    onInstallMessageIdChanged: {
        toast.show(installed.message, installed.messageUndoable ? qsTr("Undo") : "")
        toastMessageId = installed.messageId
    }

    Connections {
        target: toast

        function onActionTriggered() {
            installed.undo = window.toastMessageId
        }

        function onDismissed() {
            installed.messageGone = window.toastMessageId
        }
    }

    // A focused text field takes Ctrl+Z for itself first.
    Shortcut {
        sequences: [StandardKey.Undo]
        enabled: toast.shown && toast.hasAction
        onActivated: toast.trigger()
    }
}
