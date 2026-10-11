import QtQuick

// Make the background under it clear while it shows: otherwise the
// background's antialiased edge shows around it as a thin outline.
Rectangle {
    id: root

    property bool shown: true
    // For small marks such as a checkbox, which are lighter.
    property bool mark: false

    opacity: shown ? 1 : 0
    visible: opacity > 0

    gradient: Gradient {
        GradientStop {
            position: 0
            color: root.mark ? Theme.disabledMarkTop : Theme.disabledTop
        }

        GradientStop {
            position: 1
            color: root.mark ? Theme.disabledMarkBottom : Theme.disabledBottom
        }
    }

    Behavior on opacity {
        NumberAnimation {
            duration: Motion.fast01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }
}
