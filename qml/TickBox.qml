import QtQuick
import QtQuick.Controls.Fusion

// Not named CheckBox: Qt Quick Controls has a type of that name already.
//
// Clicking it doesn't change checked by itself: it emits clicked(), and
// whatever checked is bound to decides, e.g. a mod being turned on.
AbstractButton {
    id: root

    // Hovering what it's the checkbox for, e.g. the label or the card it's on,
    // counts as hovering it, as clicking that toggles it too.
    property bool labelHovered: false
    readonly property bool lit: enabled && (hovered || labelHovered)

    implicitWidth: 22
    implicitHeight: 22
    padding: 0

    hoverEnabled: true
    focusPolicy: Qt.TabFocus

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }

    background: Item {
        // Carbon's focus ring, a little outside the box.
        Rectangle {
            anchors {
                fill: parent
                margins: -3
            }
            radius: box.radius + 3
            color: "transparent"
            border.width: 2
            border.color: Theme.iris
            visible: root.visualFocus
        }

        Rectangle {
            id: box

            anchors.fill: parent
            radius: 5
            // A dark, see-through box while unchecked, so it shows up on
            // bright images as well as dark rows.
            // Clear while disabled, so its antialiased edge doesn't show
            // around the disabled look's.
            color: !root.enabled ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0)
                 : root.checked ? (root.lit ? Theme.lift(Theme.accent, 0.08) : Theme.accent)
                 : Qt.rgba(Theme.base.r, Theme.base.g, Theme.base.b, root.lit ? 0.7 : 0.55)
            border.width: root.checked || !root.enabled ? 0 : 1.5
            border.color: root.lit ? Theme.text : Theme.subtle

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            Behavior on border.color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            DisabledFill {
                anchors.fill: parent
                radius: parent.radius
                mark: true
                shown: !root.enabled
            }
        }
    }

    contentItem: Item {
        TintedIcon {
            id: tick

            // From 0 to 1.
            property real grown: 0

            anchors.centerIn: parent
            width: Math.round(parent.width * 0.75)
            height: width
            source: "icons/checkmark_24_filled.svg"
            // Kept while disabled, so it still shows what it will be.
            color: root.enabled ? Theme.onAccent : Theme.highlightMed
            opacity: grown > 0 ? 1 : 0
            // Grows from the corner of the tick, as Carbon's does.
            transform: Scale {
                origin.x: tick.width * 0.35
                origin.y: tick.height * 0.78
                xScale: tick.grown
                yScale: tick.grown
            }
        }

        states: State {
            name: "checked"
            when: root.checked

            PropertyChanges {
                tick.grown: 1
            }
        }

        // Leaving is quicker than arriving.
        transitions: [
            Transition {
                to: "checked"

                NumberAnimation {
                    property: "grown"
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveEntrance
                }
            },

            Transition {
                from: "checked"

                NumberAnimation {
                    property: "grown"
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveExit
                }
            }
        ]
    }
}
