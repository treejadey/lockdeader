import QtQuick
import QtQuick.Controls.Fusion

// Shown on an installed mod that's off because it changes the same files as
// mods that are on: only one can be on at a time, which is how it should
// be, so it's a quiet note like MergeNotice, not a warning. Hovering it
// names them.
Item {
    id: root

    // The mods that are on, joined; "" hides it.
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
        source: "icons/info_24_regular.svg"
        color: Theme.subtle
    }

    HoverHandler {
        id: hover
    }

    ToolTip {
        visible: hover.hovered
        delay: 500
        text: qsTr("Can’t be on at the same time as %1: they change the same files").arg(root.clashes)
        width: Math.min(implicitWidth, 360)
    }
}
