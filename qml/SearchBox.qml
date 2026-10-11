import QtQuick
import QtQuick.Controls.Fusion

// In the style of Carbon's text input.
Rectangle {
    id: root

    property alias text: input.text
    property string placeholderText

    // Emitted as the user types, not when text is set from code.
    signal textEdited()
    signal accepted()

    implicitWidth: 280
    implicitHeight: 36
    topLeftRadius: 4
    topRightRadius: 4
    color: hover.hovered && !input.activeFocus ? Qt.lighter(Theme.overlay, 1.15) : Theme.overlay

    Behavior on color {
        ColorAnimation {
            duration: Motion.fast01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    HoverHandler {
        id: hover
    }

    TextField {
        id: input

        anchors {
            left: parent.left
            right: searchButton.left
            top: parent.top
            bottom: parent.bottom
        }
        leftPadding: 12
        rightPadding: 4
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

    AbstractButton {
        id: searchButton

        anchors {
            right: parent.right
            rightMargin: 4
            verticalCenter: parent.verticalCenter
        }
        width: 28
        height: 28
        padding: 4

        hoverEnabled: true
        // Clicking it leaves the focus in the field.
        focusPolicy: Qt.NoFocus
        onClicked: root.accepted()

        scale: down ? 0.92 : 1

        Behavior on scale {
            NumberAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        HoverHandler {
            cursorShape: Qt.PointingHandCursor
        }

        background: Rectangle {
            radius: 4
            color: searchButton.hovered ? Theme.highlightMed : "transparent"

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        contentItem: Image {
            source: "icons/search_24_regular.svg"
            sourceSize: Qt.size(width, height)
            fillMode: Image.PreserveAspectFit
            opacity: searchButton.hovered ? 1 : 0.6

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
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
