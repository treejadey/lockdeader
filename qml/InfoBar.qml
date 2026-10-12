import QtQuick
import QtQuick.Controls.Fusion

// A notice that sits in the view, rather than over it, and stays until
// dealt with: for something that needs attention, such as the mods not
// reaching the game (see docs/interface-guidelines.md, Feedback). Where it
// goes is up to its parent; it takes no room while not shown.
Rectangle {
    id: root

    property string text
    // What went wrong as Go wrote it, shown on hover; "" for none.
    property string detail: ""
    // "information", "warning" or "error".
    property string severity: "information"
    // The button's label, e.g. "Retry"; "" for none.
    property string action: ""
    property bool shown: false

    signal actionTriggered()

    readonly property color edgeColor: severity === "error" ? Theme.love
                                     : severity === "warning" ? Theme.gold
                                     : Theme.foam

    implicitHeight: shown ? Math.max(label.implicitHeight, button.implicitHeight) + 16 : 0
    clip: true
    visible: height > 0
    color: Theme.overlay

    Behavior on implicitHeight {
        NumberAnimation {
            duration: Motion.moderate01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    Rectangle {
        anchors {
            left: parent.left
            top: parent.top
            bottom: parent.bottom
        }
        width: 3
        color: root.edgeColor
    }

    Label {
        id: label

        anchors {
            left: parent.left
            leftMargin: 15
            right: button.visible ? button.left : parent.right
            rightMargin: 12
            verticalCenter: parent.verticalCenter
        }
        text: root.text
        color: Theme.text
        wrapMode: Text.Wrap

        HoverHandler {
            id: hover

            enabled: root.detail !== ""
        }

        ToolTip {
            visible: hover.hovered
            delay: 500
            width: Math.min(implicitWidth, 420)
            text: root.detail
            font.family: app.monospaceFont
        }
    }

    ActionButton {
        id: button

        anchors {
            right: parent.right
            rightMargin: 8
            verticalCenter: parent.verticalCenter
        }
        visible: root.action !== ""
        text: root.action
        baseColor: Theme.highlightMed
        onClicked: root.actionTriggered()
    }
}
