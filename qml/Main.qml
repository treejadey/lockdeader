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
    title: "Lockdeader " + app.version

    // Reopens on the page the app was left on.
    Component.onCompleted: sidebar.currentIndex = uiState.page

    RowLayout {
        anchors.fill: parent
        spacing: 0

        Sidebar {
            id: sidebar

            settingsSections: settingsPage.sectionList
            currentSettingsSection: settingsPage.currentSection
            onSettingsSectionClicked: (index) => settingsPage.goToSection(index)
            onCurrentIndexChanged: uiState.page = currentIndex

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

            InstalledPage {
                onBrowseModsRequested: sidebar.currentIndex = 0
                onTakeOverRequested: takeOverDialog.open()
            }

            SettingsPage {
                id: settingsPage

                onTakeOverRequested: takeOverDialog.open()
            }
        }
    }

    TakeOverDialog {
        id: takeOverDialog
    }

    // In the overlay, over any dialog, so a dialog closing as a toast appears
    // doesn't hide it.
    Item {
        parent: Overlay.overlay
        anchors.fill: parent
        z: 1000

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
    // be passed back; 0 while it shows another bridge's.
    property int toastMessageId: 0

    // Showing a message dismisses the one before, which still has the old id.
    onInstallMessageIdChanged: {
        toast.show(installed.message, installed.messageUndoable ? qsTr("Undo") : "")
        toastMessageId = installed.messageId
    }

    readonly property int launcherMessageId: launcher.messageId
    // The launcher's message offers to open the settings.
    property bool toastOpensSettings: false

    onLauncherMessageIdChanged: {
        toastOpensSettings = launcher.messageAction === "settings"
        toast.show(launcher.message, toastOpensSettings ? qsTr("Settings") : "")
        toastMessageId = 0
    }

    Connections {
        target: toast

        function onActionTriggered() {
            if (window.toastMessageId !== 0)
                installed.undo = window.toastMessageId
            else if (window.toastOpensSettings)
                sidebar.currentIndex = 2
        }

        function onDismissed() {
            if (window.toastMessageId !== 0)
                installed.messageGone = window.toastMessageId
        }
    }

    // A focused text field takes Ctrl+Z for itself first. Only Undo is
    // undoing; the toast's other actions aren't.
    Shortcut {
        sequences: [StandardKey.Undo]
        enabled: toast.shown && toast.hasAction && window.toastMessageId !== 0
        onActivated: toast.trigger()
    }
}
