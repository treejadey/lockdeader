import QtQuick
import QtQuick.Controls.Fusion

// Shown on an installed mod that changes the same files as other installed
// mods, which can't be on at the same time; hovering it names them.
Item {
    id: root

    // The other mods' names, joined; "" hides the warning.
    property string clashes
    // Over a mod's picture, it sits on a badge so it shows on any picture.
    property bool onPicture: false

    implicitWidth: onPicture ? 22 : 16
    implicitHeight: implicitWidth
    visible: clashes !== ""

    // Looks like an unchecked TickBox, which it sits across the card from.
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
        source: "icons/warning_24_regular.svg"
        color: Theme.gold
    }

    HoverHandler {
        id: hover
    }

    ToolTip {
        visible: hover.hovered
        delay: 500
        text: qsTr("Changes the same files as %1").arg(root.clashes)
        width: Math.min(implicitWidth, 360)
    }
}
