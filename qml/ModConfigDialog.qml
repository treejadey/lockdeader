import QtQuick
import QtQuick.Controls.Fusion

AppDialog {
    id: root

    // As in installed.configuration: {key, name, loading, error, errorDetail, options},
    // where options are {id, name, detail, size, added, picked, installed}
    // objects.
    property var configuration: ({})
    readonly property var variants: configuration.options ?? []

    property var selectedIds: []
    readonly property bool onlyOne: selectedIds.length === 1
    // The picks differ from what the mod has.
    readonly property bool changed: {
        const enabled = variants.filter(v => v.picked).map(v => v.id)

        return enabled.length !== selectedIds.length || enabled.some(id => !isSelected(id))
    }

    signal changeRequested(string key, var ids)

    function isSelected(id) {
        return selectedIds.indexOf(id) >= 0
    }

    function toggle(id) {
        if (!isSelected(id))
            selectedIds = selectedIds.concat([id])
        else if (!onlyOne)
            selectedIds = selectedIds.filter(other => other !== id)
    }

    function reset() {
        selectedIds = variants.filter(v => v.picked).map(v => v.id)
    }

    // The list can change while it's open, e.g. once the mod's other files
    // on GameBanana have been looked up: picks already made are kept.
    onVariantsChanged: {
        if (changed)
            selectedIds = selectedIds.filter(id => variants.some(v => v.id === id))
        else
            reset()
    }

    onAboutToShow: reset()

    titlePrefix: configuration.name ?? ""
    title: qsTr("Configuration")

    FilePickerList {
        files: root.variants
        selectedIds: root.selectedIds
        keepOne: true
        onToggled: (id) => root.toggle(id)
    }

    // While the mod's files on GameBanana are being looked up, only its
    // installed options are listed.
    Row {
        x: 2
        width: parent.width - 2
        spacing: 8
        visible: root.configuration.loading === true || (root.configuration.error ?? "") !== ""

        BusyIndicator {
            anchors.verticalCenter: parent.verticalCenter
            width: 18
            height: 18
            running: root.configuration.loading === true
            visible: running
        }

        ErrorLabel {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - (root.configuration.loading === true ? 26 : 0)
            text: root.configuration.loading === true ? qsTr("Looking for other versions on GameBanana…")
                : qsTr("Couldn’t look for other versions: %1").arg(root.configuration.error)
            detail: root.configuration.loading === true ? "" : root.configuration.errorDetail ?? ""
            color: root.configuration.loading === true ? Theme.subtle : Theme.love
            wrapMode: Text.NoWrap
            elide: Text.ElideRight
        }
    }

    // Hidden with opacity, so it keeps its room and the dialog doesn't change
    // size.
    Label {
        x: 2
        width: parent.width - 2
        text: qsTr("At least one variant must be selected.")
        color: Theme.subtle
        font.weight: Font.DemiBold
        wrapMode: Text.Wrap
        opacity: root.onlyOne ? 1 : 0

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    buttons: ActionButton {
        width: Math.max(implicitWidth, 218)
        implicitHeight: 44
        text: qsTr("Change Configuration")
        baseColor: Theme.accent
        textColor: Theme.textOnAccent
        enabled: root.changed
        onClicked: {
            root.changeRequested(root.configuration.key, root.selectedIds)
            root.close()
        }
    }
}
