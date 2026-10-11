import QtQuick
import QtQuick.Controls.Fusion

Rectangle {
    id: root

    property string text
    property string description
    property bool monospaceDescription: false
    // Other words that should find the setting when searched for.
    property string keywords
    // Makes the whole row hover and emit rowPressed, e.g. to flip its switch.
    property bool interactive: false

    default property alias control: slot.data

    // Not emitted for presses on the control, which handles its own.
    signal rowPressed()

    // needle must be in lower case; "" matches every row.
    function matches(needle) {
        return needle === ""
            || (text + "\n" + description + "\n" + keywords).toLowerCase().includes(needle)
    }

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

    // Acts on press, like the switch.
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
