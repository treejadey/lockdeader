import QtQuick
import QtQuick.Controls.Fusion

// A checkbox that behaves like Carbon's: a small rounded square that fills in
// foam, with a dark tick, while it's checked. While it's disabled it takes
// the app's disabled look (see DisabledFill), in light grey, keeping its tick
// if it has one, so it still shows what it will be.
// Like Carbon's, it shows a ring just outside the box while focused with the
// keyboard, and Space toggles it. (Not named CheckBox: Qt Quick Controls has
// a type of that name already.)
//
// Clicking it doesn't change checked by itself: it emits clicked(), and
// whatever checked is bound to decides, e.g. a mod being turned on.
//
// Motion is productive, as for Carbon's checkbox: the box changes colour over
// fast-01. The tick grows from its corner over fast-02 with an entrance
// curve, and shrinks back into it over fast-01 with an exit curve, as
// leaving should be quicker than arriving.
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
        // Carbon's focus ring: 2px, a little outside the box.
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
            color: !root.enabled ? Qt.rgba(Theme.foam.r, Theme.foam.g, Theme.foam.b, 0)
                 : root.checked ? (root.lit ? Qt.lighter(Theme.foam, 1.08) : Theme.foam)
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
        // Fluent's checkmark, in whatever colour the box needs.
        TintedIcon {
            id: tick

            // How far the tick has grown, from 0 to 1.
            property real grown: 0

            anchors.centerIn: parent
            width: Math.round(parent.width * 0.75)
            height: width
            source: "icons/checkmark_24_filled.svg"
            color: root.enabled ? Theme.base : Theme.highlightMed
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
