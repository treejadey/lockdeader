import QtQuick
import QtQuick.Controls.Fusion

// A text field in the style of Carbon's text input, as SearchBox is.
Rectangle {
    id: root

    property alias text: input.text
    property string placeholderText

    // Emitted as the user types, not when text is set from code.
    signal textEdited()
    signal accepted()

    implicitWidth: 260
    implicitHeight: 36
    topLeftRadius: 4
    topRightRadius: 4
    color: hover.hovered && !input.activeFocus ? Qt.lighter(Theme.highlightLow, 1.15) : Theme.highlightLow

    Behavior on color {
        ColorAnimation {
            duration: Motion.fast01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    HoverHandler {
        id: hover

        cursorShape: Qt.IBeamCursor
    }

    TextField {
        id: input

        anchors.fill: parent
        leftPadding: 12
        rightPadding: 12
        verticalAlignment: TextInput.AlignVCenter
        background: null

        placeholderText: root.placeholderText
        color: Theme.text
        placeholderTextColor: Theme.subtle
        selectionColor: Theme.highlightHigh
        selectedTextColor: Theme.text
        selectByMouse: true

        onTextEdited: root.textEdited()
        onAccepted: root.accepted()
    }

    // The line along the bottom.
    Rectangle {
        anchors {
            left: parent.left
            right: parent.right
            bottom: parent.bottom
        }
        height: input.activeFocus ? 2 : 1
        color: input.activeFocus ? Theme.foam : Theme.highlightHigh

        Behavior on height {
            NumberAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }
}
