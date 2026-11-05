import QtQuick
import QtQuick.Controls.Fusion

// Shown on an installed mod that shares sound tables with other mods that
// are on, when merging them left some of its changes out; hovering it says
// which mods. Quieter than ClashWarning, as nothing is wrong.
Item {
    id: root

    // The other mods' names, joined; "" hides it.
    property string mergedWith
    // Over a mod's picture, it sits on a badge so it shows on any picture.
    property bool onPicture: false

    implicitWidth: onPicture ? 22 : 16
    implicitHeight: implicitWidth
    visible: mergedWith !== ""

    // The same badge as ClashWarning's.
    Rectangle {
        anchors.fill: parent
        visible: root.onPicture
        radius: 5
        color: Qt.rgba(Theme.base.r, Theme.base.g, Theme.base.b, 0.55)
        border.width: 1.5
        border.color: Theme.subtle
    }

    TintedIcon {
        anchors.centerIn: parent
        width: 16
        height: 16
        source: "icons/info_24_regular.svg"
        color: Theme.subtle
    }

    HoverHandler {
        id: hover
    }

    ToolTip {
        visible: hover.hovered
        delay: 500
        text: qsTr("Some of its sound changes are left out so it works alongside %1").arg(root.mergedWith)
        width: Math.min(implicitWidth, 360)
    }
}
