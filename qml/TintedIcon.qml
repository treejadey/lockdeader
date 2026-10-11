import QtQuick
import QtQuick.Effects

// The SVG's own fill doesn't matter.
Item {
    id: root

    property url source
    property color color: Theme.text

    Rectangle {
        anchors.fill: parent
        color: root.color

        layer.enabled: true
        layer.effect: MultiEffect {
            maskEnabled: true
            maskSource: shape
            // Softens the mask's edge, so the icon is antialiased.
            maskThresholdMin: 0.5
            maskSpreadAtMin: 1.0
        }
    }

    Image {
        id: shape

        anchors.fill: parent
        source: root.source
        sourceSize: Qt.size(width, height)
        visible: false
        layer.enabled: true
    }
}
