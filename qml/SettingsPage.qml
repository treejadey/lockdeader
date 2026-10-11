import QtQuick
import QtQuick.Controls.Fusion

// The sidebar lists the sections (see SettingsNav): currentSection is the one
// being read, and goToSection() scrolls to one.
Item {
    id: root

    readonly property string query: topBar.searchText
    readonly property var sectionItems: Array.from(sections.children)
    readonly property bool anyMatches: sectionItems.some(section => section.hasMatches)
    // {title, shown} for each section; shown is false while a search hides it.
    // Asks to take the mods from other tools over (see TakeOverDialog).
    signal takeOverRequested()

    readonly property var sectionList: sectionItems.map(section => ({
        title: section.title,
        shown: section.hasMatches,
    }))

    // The section being read: the last one whose heading has scrolled up to
    // the top of the page (within 64px of it), or the last one once scrolled
    // to the bottom, which could otherwise never get there. After
    // goToSection(), the section gone to, until the user scrolls themselves:
    // it may be too near the bottom to reach the top.
    readonly property int currentSection: {
        if (goneTo >= 0 && sectionItems[goneTo].hasMatches)
            return goneTo

        const top = flickable.contentY + 64
        const atBottom = flickable.contentHeight > flickable.height && flickable.atYEnd
        let current = -1

        for (let i = 0; i < sectionItems.length; i++) {
            const section = sectionItems[i]
            if (!section.hasMatches)
                continue

            if (current < 0 || atBottom || sectionTop(section) <= top)
                current = i
        }

        return current
    }

    // The section last gone to with goToSection(), until the user scrolls.
    property int goneTo: -1
    // As on the mods page: until the scroll position is restored, it isn't
    // saved.
    property bool scrollRestored: false

    function restoreScroll() {
        if (scrollRestored || !visible)
            return

        scrollRestored = true
        flickable.contentY = Math.min(uiState.settingsY, Math.max(0, flickable.contentHeight - flickable.height))
    }

    function saveScroll() {
        if (scrollRestored)
            uiState.settingsY = Math.round(flickable.contentY)
    }

    // Once laid out, so the page is as tall as it will be.
    Component.onCompleted: Qt.callLater(restoreScroll)

    onVisibleChanged: {
        Qt.callLater(restoreScroll)

        // Deadlock may have been installed since the app started.
        if (visible && settings.gameFolder === "")
            settings.findGame = true
    }

    function sectionTop(section) {
        return sections.y + section.y
    }

    // Scrolls a section's heading to the top of the page, or as near as it
    // can get.
    function goToSection(index) {
        const section = sectionItems[index]
        if (!section || !section.hasMatches)
            return

        wheel.stop()

        const bottom = Math.max(0, flickable.contentHeight - flickable.height)
        const target = Math.min(bottom, sectionTop(section) - sections.anchors.topMargin)

        goneTo = index

        scroll.stop()
        scroll.from = flickable.contentY
        scroll.to = target
        // Further takes longer, as Carbon advises; a page or more takes the
        // longest.
        scroll.duration = Math.abs(target - flickable.contentY) < flickable.height
            ? Motion.moderate02 : Motion.slow01
        scroll.start()
    }

    TopBar {
        id: topBar

        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
        }
        crumbs: [{ name: qsTr("Settings"), depth: 0 }]
        searchPlaceholder: qsTr("Search Settings")
    }

    Flickable {
        id: flickable

        anchors {
            top: topBar.bottom
            bottom: parent.bottom
            left: parent.left
            right: scrollBar.left
        }
        contentHeight: sections.implicitHeight + sections.anchors.topMargin + 24
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        ScrollBar.vertical: scrollBar

        // Scrolling by any other means than goToSection() lets the sections
        // being read take over again.
        onContentYChanged: {
            if (!scroll.running)
                root.goneTo = -1

            root.saveScroll()
        }
        // The user takes over from goToSection() by dragging...
        onMovementStarted: scroll.stop()

        NumberAnimation {
            id: scroll

            target: flickable
            property: "contentY"
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }

        Column {
            id: sections

            anchors {
                top: parent.top
                topMargin: 16
                left: parent.left
                leftMargin: 16
                right: parent.right
                rightMargin: 4
            }
            spacing: 32

            SettingsSection {
                id: gameSection

                width: parent.width
                title: qsTr("Game")
                query: root.query

                SettingRow {
                    text: qsTr("Deadlock folder")
                    description: settings.gameFolder !== "" ? settings.gameFolder
                        : qsTr("Not found in Steam’s libraries. Choose the folder Steam installed it in.")
                    monospaceDescription: settings.gameFolder !== ""
                    error: settings.gameFolderError
                    keywords: qsTr("game location install path directory steam library")

                    Row {
                        spacing: 6

                        ActionButton {
                            visible: settings.gameFolder !== ""
                            text: qsTr("Open Folder")
                            baseColor: Theme.highlightMed
                            onClicked: Qt.openUrlExternally(settings.gameFolderUrl)
                        }

                        // Go opens the system's folder dialog, and checks
                        // the folder picked: one that isn't Deadlock's is
                        // turned down, with gameFolderError.
                        ActionButton {
                            text: settings.gameFolder !== "" ? qsTr("Change…") : qsTr("Choose…")
                            baseColor: Theme.highlightMed
                            enabled: !settings.choosingGameFolder
                            onClicked: settings.chooseGameFolder = true
                        }
                    }
                }

                SettingRow {
                    text: qsTr("Launch options")
                    description: qsTr("Added to the game’s command line when Lockdeader starts it")
                    keywords: qsTr("command line arguments parameters flags startup")

                    TextBox {
                        text: settings.launchOptions
                        onTextEdited: settings.launchOptions = text
                    }
                }

                SettingRow {
                    text: qsTr("Mods from other tools")
                    description: otherMods.count === 0
                        ? qsTr("None found in the game’s addons folder")
                        : otherMods.count === 1
                        ? qsTr("One mod in the game’s addons folder, from Deadlock Mod Manager or put there by hand. Lockdeader can take it over, and then turn it on and off, update and uninstall it.")
                        : qsTr("%1 mods in the game’s addons folder, from Deadlock Mod Manager or put there by hand. Lockdeader can take them over, and then turn them on and off, update and uninstall them.").arg(otherMods.count)
                    keywords: qsTr("deadlock mod manager dmm addons folder import take over migrate port third party")

                    ActionButton {
                        text: qsTr("Take Over…")
                        baseColor: Theme.highlightMed
                        enabled: otherMods.count > 0 && !otherMods.takingOver
                        onClicked: root.takeOverRequested()
                    }
                }

                SettingToggle {
                    text: qsTr("Also load mods installed by other tools")
                    description: qsTr("Mods put in the game’s addons folder by hand or by another mod manager load after Lockdeader’s when it starts Deadlock with mods")
                    keywords: qsTr("addons folder deadlock mod manager dmm manual external other")
                    checked: settings.loadOtherMods
                    onToggled: settings.loadOtherMods = checked
                }
            }

            SettingsSection {
                id: modViewer

                width: parent.width
                title: qsTr("Mod Viewer")
                query: root.query

                SettingToggle {
                    text: qsTr("NSFW mods")
                    keywords: qsTr("not safe for work adult mature nudity content ratings")
                    checked: settings.showNsfw
                    onToggled: settings.showNsfw = checked
                }
            }

            SettingsSection {
                id: advanced

                width: parent.width
                title: qsTr("Advanced")
                query: root.query

                SettingRow {
                    text: qsTr("Data folder")
                    description: settings.dataDir
                    monospaceDescription: true
                    keywords: qsTr("data directory location cache files backups open")

                    ActionButton {
                        text: qsTr("Open Folder")
                        baseColor: Theme.highlightMed
                        onClicked: Qt.openUrlExternally(settings.dataDirUrl)
                    }
                }
            }
        }
    }

    SmoothWheel {
        id: wheel

        anchors.fill: flickable
        flickable: flickable
        // ...or by wheeling.
        onWheel: scroll.stop()
    }

    ScrollBar {
        id: scrollBar

        anchors {
            top: topBar.bottom
            bottom: parent.bottom
            right: parent.right
        }
        onPressedChanged: if (pressed) {
            wheel.stop()
            scroll.stop()
        }
    }

    Label {
        anchors.centerIn: flickable
        visible: !root.anyMatches
        text: qsTr("No Settings Found")
        color: Theme.subtle
    }
}
