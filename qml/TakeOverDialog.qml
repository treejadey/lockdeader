import QtQuick
import QtQuick.Controls.Fusion

// Asks before taking the mods from other tools over, and for mods with
// several files on GameBanana, which to download: the same way the install
// dialog picks files, one list per mod. The dialog grows with the lists, and
// scrolls as a whole (see AppDialog).
AppDialog {
    id: root

    readonly property int count: otherMods.count
    readonly property var choices: otherMods.choices
    // The file ids picked, by mod key.
    property var picks: ({})

    function resetPicks() {
        const next = {}

        for (const choice of root.choices)
            next[choice.key] = root.picks[choice.key] ?? [choice.picked]

        root.picks = next
    }

    function toggle(key, id) {
        const current = root.picks[key] ?? []
        const next = Object.assign({}, root.picks)

        next[key] = current.indexOf(id) >= 0 ? current.filter(other => other !== id) : current.concat([id])
        root.picks = next
    }

    onAboutToShow: {
        root.picks = {}
        resetPicks()
    }

    // Lookups can finish while it's open.
    onChoicesChanged: resetPicks()

    title: qsTr("Take Over Mods From Other Tools")
    cardWidth: 560

    Label {
        width: parent.width
        wrapMode: Text.Wrap
        color: Theme.text
        text: root.count === 1
            ? qsTr("The mod in the game’s addons folder will be moved into Lockdeader, which will then turn it on and off, update and uninstall it. Deadlock Mod Manager won’t see it any more.")
            : qsTr("The %1 mods in the game’s addons folder will be moved into Lockdeader, which will then turn them on and off, update and uninstall them. Deadlock Mod Manager won’t see them any more.").arg(root.count)
    }

    Label {
        width: parent.width
        wrapMode: Text.Wrap
        color: Theme.subtle
        text: qsTr("Mods GameBanana has are downloaded from it again, so they’re no different from ones installed here. Files it doesn’t know are kept as they are, by their file names.")
    }

    Label {
        width: parent.width
        visible: root.choices.length > 0
        wrapMode: Text.Wrap
        color: Theme.text
        text: root.choices.length === 1
            ? qsTr("One of them has several files on GameBanana. Pick which to download:")
            : qsTr("%1 of them have several files on GameBanana. Pick which to download:").arg(root.choices.length)
    }

    Repeater {
        model: root.choices

        Column {
            required property var modelData

            width: parent.width
            spacing: 6

            Label {
                width: parent.width
                text: modelData.name
                color: Theme.text
                font.weight: Font.DemiBold
                elide: Text.ElideRight
            }

            FilePickerList {
                files: modelData.files
                detailKey: "description"
                selectedIds: root.picks[modelData.key] ?? []
                keepOne: true
                // The dialog scrolls, not the list.
                visibleRows: 1000
                onToggled: (id) => root.toggle(modelData.key, id)
            }
        }
    }

    buttons: [
        ActionButton {
            text: qsTr("Cancel")
            onClicked: root.close()
        },
        ActionButton {
            text: root.count === 1 ? qsTr("Take Over the Mod") : qsTr("Take Over %1 Mods").arg(root.count)
            baseColor: Theme.accent
            textColor: Theme.textOnAccent
            // Not before it's known which mods are on GameBanana.
            enabled: root.count > 0 && !otherMods.takingOver && !otherMods.lookingUp
            onClicked: {
                otherMods.takeOver = Object.keys(root.picks).map(key => ({ key: key, ids: root.picks[key] }))
                root.close()
            }
        }
    ]
}
