import QtQuick
import QtQuick.Effects

// Make the placeholder transparent to show something else underneath while
// the image loads.
Item {
    id: root

    property alias source: image.source
    readonly property alias status: image.status
    property real radius: 6
    property color placeholderColor: Theme.overlay
    property bool blurred: false
    // Which part of a picture of another shape stays when it's cropped.
    property alias verticalAlignment: image.verticalAlignment

    Item {
        anchors.fill: parent

        layer.enabled: true
        layer.effect: MultiEffect {
            maskEnabled: true
            maskSource: mask
            // Soften the mask edge so the corners are antialiased.
            maskThresholdMin: 0.5
            maskSpreadAtMin: 1.0
        }

        Rectangle {
            anchors.fill: parent
            color: root.placeholderColor
        }

        Image {
            id: image

            anchors.fill: parent
            sourceSize: Qt.size(width, height)
            asynchronous: true
            fillMode: Image.PreserveAspectCrop
            opacity: status === Image.Ready ? 1 : 0

            layer.enabled: root.blurred
            layer.effect: MultiEffect {
                blurEnabled: true
                blur: 1.0
                blurMax: 32
                // Without padding, the blur repeats the edge pixels instead
                // of fading into transparency around the border.
                autoPaddingEnabled: false
            }

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }
    }

    Item {
        id: mask

        anchors.fill: parent
        layer.enabled: true
        visible: false

        Rectangle {
            anchors.fill: parent
            radius: root.radius
            color: "black"
        }
    }
}
