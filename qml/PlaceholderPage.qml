import QtQuick
import QtQuick.Controls.Fusion

// Stand-in for pages that haven't been built yet.
Item {
    id: root

    property string title
    property string text

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
