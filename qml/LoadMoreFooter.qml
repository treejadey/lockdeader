import QtQuick
import QtQuick.Controls.Fusion

// The end of a lazily loaded list or grid: a spinner while the next page is
// on its way, or the error and a Retry button if it failed. It's the same
// height whether or not anything is showing, so the content doesn't jump when
// loading starts or stops.
Item {
    id: root

    property bool loading: false
    property string error: ""

    signal retry()

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
            text: "Couldn't load more mods: " + root.error
            elide: Text.ElideRight
        }

        Button {
            id: retryButton
            text: "Retry"
            onClicked: root.retry()
        }
    }
}
