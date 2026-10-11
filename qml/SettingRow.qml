import QtQuick
import QtQuick.Controls.Fusion

// One setting on the settings page: its name, and optionally a description
// under it, at the left, and the control that changes it (its children) at
// the right.
Rectangle {
    id: root

    property string text
    property string description
    // Sets the description in monospace, e.g. for a path.
    property bool monospaceDescription: false
    // Other words that should find the setting when searched for.
    property string keywords
    // Makes the whole row hover and emit rowPressed, e.g. to flip its switch.
    property bool interactive: false

    default property alias control: slot.data

    // Emitted when an interactive row is pressed somewhere other than its
    // control, which handles its own presses.
    signal rowPressed()

    // Whether the row should show for a search, in lower case; "" matches
    // every row.
    function matches(needle) {
        return needle === ""
            || (text + "\n" + description + "\n" + keywords).toLowerCase().includes(needle)
    }

    // Rows fill their section.
    width: parent ? parent.width : implicitWidth
    implicitHeight: Math.max(44, labels.implicitHeight + 20, slot.childrenRect.height + 16)
    radius: 6
    color: root.interactive && hover.hovered ? Qt.lighter(Theme.overlay, 1.15) : Theme.overlay

    Behavior on color {
        ColorAnimation {
            duration: Motion.fast01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    HoverHandler {
        id: hover
        enabled: root.interactive
        cursorShape: Qt.PointingHandCursor
    }

    // Productive, like the switch: acts on press.
    TapHandler {
        enabled: root.interactive
        onPressedChanged: if (pressed) root.rowPressed()
    }

    Column {
        id: labels

        anchors {
            left: parent.left
            leftMargin: 16
            right: slot.left
            rightMargin: 16
            verticalCenter: parent.verticalCenter
        }
        spacing: 2

        Label {
            width: parent.width
            text: root.text
            color: Theme.text
            elide: Text.ElideRight
        }

        Label {
            width: parent.width
            visible: text !== ""
            text: root.description
            color: Theme.subtle
            font.family: root.monospaceDescription ? app.monospaceFont : Qt.application.font.family
            font.pixelSize: 12
            wrapMode: root.monospaceDescription ? Text.WrapAnywhere : Text.Wrap
        }
    }

    Item {
        id: slot

        anchors {
            right: parent.right
            rightMargin: 10
            verticalCenter: parent.verticalCenter
        }
        width: childrenRect.width
        height: childrenRect.height
    }
}
