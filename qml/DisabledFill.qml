import QtQuick

// How a disabled control's surface looks, the same across the app: a slight
// gradient from brighter at the top to darker at the bottom. It goes over the
// control's own background, matching its shape, and fades in while the
// control is disabled, and back out when it isn't (fast-01, standard easing,
// as for Carbon's other micro-interactions). Make the background under it
// clear while it shows: otherwise the background's antialiased edge shows
// around it as a thin outline.
Rectangle {
    id: root

    // Whether it's showing, e.g. while the control is disabled.
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
