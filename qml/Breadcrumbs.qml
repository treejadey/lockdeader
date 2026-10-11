import QtQuick
import QtQuick.Controls.Fusion

// Where the user is, e.g. "Skins > Abrams": the current place stands out in
// the text colour, and the places above it are subtle, and can be clicked to
// go back up to them. The arrows between them are dim and can't be clicked.
// When they don't all fit, the names above the current place are cut short
// with "…" first.
Item {
    id: root

    // {name, depth} objects, from the top down. depth is what crumbClicked
    // reports for that crumb.
    property var crumbs: []

    signal crumbClicked(int depth)

    readonly property int spacing: 6
    readonly property int arrowSize: 14

    // How wide each crumb's name gets to be.
    readonly property var widths: {
        const count = crumbs.length
        if (count === 0)
            return []
        const natural = crumbs.map((crumb, i) => Math.ceil(
            (i === count - 1 ? currentMetrics : otherMetrics).advanceWidth(crumb.name)))
        let room = Math.max(0, width - (count - 1) * (2 * spacing + arrowSize))

        const widths = natural.slice()
        widths[count - 1] = Math.min(natural[count - 1], room)
        room -= widths[count - 1]
        // Share what's left between the others, in proportion to their names.
        const others = natural.slice(0, -1).reduce((sum, w) => sum + w, 0)
        if (others > room) {
            for (let i = 0; i < count - 1; i++)
                widths[i] = Math.floor(natural[i] * room / others)
        }
        return widths
    }

    implicitHeight: row.implicitHeight
    clip: true

    FontMetrics {
        id: otherMetrics
    }

    FontMetrics {
        id: currentMetrics
        font.weight: Font.Medium
    }

    Row {
        id: row

        anchors.verticalCenter: parent.verticalCenter
        spacing: root.spacing

        Repeater {
            model: root.crumbs

            Row {
                id: crumb

                required property var modelData
                required property int index
                readonly property bool current: index === root.crumbs.length - 1

                spacing: root.spacing

                Image {
                    anchors.verticalCenter: parent.verticalCenter
                    width: root.arrowSize
                    height: root.arrowSize
                    visible: crumb.index > 0
                    source: "icons/chevron_right_24_regular.svg"
                    sourceSize: Qt.size(width, height)
                    opacity: 0.35
                }

                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    width: root.widths[crumb.index] ?? 0
                    text: crumb.modelData.name
                    elide: Text.ElideRight
                    color: crumb.current || crumbHover.hovered ? Theme.text : Theme.subtle
                    font.weight: crumb.current ? Font.Medium : Font.Normal

                    Behavior on color {
                        ColorAnimation {
                            duration: Motion.fast01
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.productiveStandard
                        }
                    }

                    HoverHandler {
                        id: crumbHover
                        enabled: !crumb.current
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        enabled: !crumb.current
                        onTapped: root.crumbClicked(crumb.modelData.depth)
                    }
                }
            }
        }
    }
}
