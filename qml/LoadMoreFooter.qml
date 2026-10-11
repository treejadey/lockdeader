import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    property bool loading: false
    property string error: ""

    signal retry()

    // The same whether or not anything is showing, so the content doesn't jump
    // when loading starts or stops.
    height: 56

    BusyIndicator {
        anchors.centerIn: parent
        running: root.loading
    }

    Row {
        anchors.centerIn: parent
        spacing: 12
        visible: root.error !== ""

        Label {
            anchors.verticalCenter: parent.verticalCenter
            width: Math.min(implicitWidth, root.width - retryButton.width - parent.spacing - 32)
            text: qsTr("Couldn’t load more mods: %1").arg(root.error)
            elide: Text.ElideRight
        }

        Button {
            id: retryButton

            text: qsTr("Retry")
            onClicked: root.retry()
        }
    }
}
