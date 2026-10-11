import QtQuick

ListView {
    id: root

    // {full, thumbnail, caption} objects.
    property var images: []
    property int thumbnailWidth: 88
    property int thumbnailHeight: 56
    property int selectedIndex: -1

    signal activated(int index)

    implicitHeight: thumbnailHeight
    orientation: ListView.Horizontal
    spacing: 6
    clip: true
    boundsBehavior: Flickable.StopAtBounds
    model: images

    // ListView resets currentIndex itself whenever the model changes, which
    // would break a plain binding, so it's copied over explicitly instead.
    currentIndex: -1
    onSelectedIndexChanged: currentIndex = selectedIndex
    onCountChanged: currentIndex = selectedIndex

    // Keep the current thumbnail near the middle where possible.
    highlightRangeMode: ListView.ApplyRange
    preferredHighlightBegin: (width - thumbnailWidth) / 2
    preferredHighlightEnd: (width + thumbnailWidth) / 2
    highlightMoveDuration: Motion.moderate01

    delegate: Item {
        id: thumb

        required property var modelData
        required property int index
        readonly property bool current: index === root.selectedIndex

        width: root.thumbnailWidth
        height: root.thumbnailHeight

        RoundedImage {
            anchors.fill: parent
            source: thumb.modelData.thumbnail
            opacity: root.selectedIndex < 0 || thumb.current || hover.hovered ? 1 : 0.55

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        Rectangle {
            anchors.fill: parent
            visible: thumb.current
            radius: 6
            color: "transparent"
            border.color: Theme.gold
            border.width: 2
        }

        HoverHandler {
            id: hover

            cursorShape: Qt.PointingHandCursor
        }

        TapHandler {
            onTapped: root.activated(thumb.index)
        }
    }
}
