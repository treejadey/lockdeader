import QtQuick
import QtQuick.Controls.Fusion

// Its search field doesn't search anything yet.
Item {
    id: root

    property string title
    property string text
    property string searchPlaceholder

    TopBar {
        id: topBar

        searchPlaceholder: root.searchPlaceholder
        anchors {
            top: parent.top
            left: parent.left
            right: parent.right
        }
    }

    Column {
        anchors.centerIn: parent
        spacing: 6

        Label {
            anchors.horizontalCenter: parent.horizontalCenter
            text: root.title
            color: Theme.text
            font.pixelSize: 18
            font.weight: Font.DemiBold
        }

        Label {
            anchors.horizontalCenter: parent.horizontalCenter
            text: root.text
            color: Theme.subtle
        }
    }
}
