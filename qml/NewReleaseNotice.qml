import QtQuick
import QtQuick.Controls.Fusion

// Shown on an installed mod that has files on GameBanana uploaded after the
// ones installed, most likely new releases of it. Unlike ClashNotice and
// MergeNotice it asks for something, so it's in the accent colour, and
// clicking it opens the mod's configuration to pick one.
AbstractButton {
    id: root

    // How many; 0 hides it.
    property int count
    // Over a mod's picture, it sits on a badge so it shows on any picture.
    property bool onPicture: false

    implicitWidth: onPicture ? 22 : 16
    implicitHeight: implicitWidth
    visible: count > 0
    padding: 0
    hoverEnabled: true
    focusPolicy: Qt.TabFocus

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }

    ToolTip.visible: hovered
    ToolTip.delay: 500
    ToolTip.text: qsTr("%n new release on GameBanana: choose which version to use",
        "%n new releases on GameBanana: choose which version to use", count)

    // Over a picture, ClashNotice's badge in the accent colour.
    background: Rectangle {
        visible: root.onPicture || root.visualFocus
        radius: root.onPicture ? 5 : 4
        color: !root.onPicture ? "transparent"
             : root.hovered ? Theme.lift(Theme.accent, 0.15)
             : Theme.accent
        border.width: root.onPicture || root.visualFocus ? 1 : 0
        border.color: root.visualFocus ? Theme.iris : Theme.badgeOutline
    }

    contentItem: Item {
        TintedIcon {
            anchors.centerIn: parent
            width: 16
            height: 16
            source: "icons/arrow_circle_up_24_regular.svg"
            color: root.onPicture ? Theme.textOnAccent
                 : root.hovered ? Theme.lift(Theme.accent, 0.25)
                 : Theme.accent
        }
    }
}
