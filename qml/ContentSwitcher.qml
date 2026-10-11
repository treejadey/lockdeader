import QtQuick
import QtQuick.Controls.Fusion

// Picks one of a few options, shown as icons in a pill, like Carbon's icon-only
// content switcher. A circle sits behind the picked option. Arrow keys move
// between the options while one has focus.
//
// Motion is productive: picking an option happens on press. Its icon fills in
// and brightens straight away while the old one goes back to its outline
// (fast-01), then the circle slides over to it (moderate-01, standard easing).
// On the way, the circle fades towards the pill's colour, and comes back to
// full strength as it arrives, so it reads as one thing moving rather than
// two options changing places.
Rectangle {
    id: root

    // {icon, text} objects: icon is the outlined (_regular) icon, swapped for
    // the filled one while that option is picked; text is its tooltip.
    property var options: []
    property int currentIndex: 0

    // Emitted when the user picks an option, after currentIndex has changed.
    signal activated(int index)

    readonly property int optionSize: 28
    readonly property int padding: 4
    readonly property int spacing: 4

    function pick(index) {
        if (index < 0 || index >= options.length || index === currentIndex)
            return
        currentIndex = index
        activated(index)
    }

    // Blends colour a towards b by amount, from 0 to 1.
    function mix(a, b, amount) {
        return Qt.rgba(a.r + (b.r - a.r) * amount, a.g + (b.g - a.g) * amount,
                       a.b + (b.b - a.b) * amount, a.a + (b.a - a.a) * amount)
    }

    implicitWidth: options.length * optionSize + Math.max(0, options.length - 1) * spacing + 2 * padding
    implicitHeight: 36
    radius: height / 2
    color: Theme.overlay

    // Where the circle is: travelling from one option to another (by index;
    // from can be partway, if it was still moving) as travel goes from 0 to 1.
    QtObject {
        id: indicator

        property real from: 0
        property real to: 0
        property real travel: 1
        readonly property real position: from + (to - from) * travel
    }

    Component.onCompleted: indicator.from = indicator.to = currentIndex

    onCurrentIndexChanged: {
        indicator.from = indicator.position
        indicator.to = currentIndex
        indicator.travel = 0
        move.restart()
    }

    SequentialAnimation {
        id: move

        // The icons change first...
        PauseAnimation {
            duration: Motion.fast01
        }
        // ...then the circle follows.
        NumberAnimation {
            target: indicator
            property: "travel"
            from: 0
            to: 1
            duration: Motion.moderate01
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveStandard
        }
    }

    // The circle behind the picked option. Declared before the options, so
    // it's drawn under them.
    Rectangle {
        x: root.padding + indicator.position * (root.optionSize + root.spacing)
        anchors.verticalCenter: parent.verticalCenter
        width: root.optionSize
        height: root.optionSize
        radius: width / 2
        // Fades most of the way into the pill halfway between options.
        color: root.mix(Theme.highlightMed, root.color, 0.6 * Math.sin(Math.PI * indicator.travel))
    }

    Row {
        anchors {
            left: parent.left
            leftMargin: root.padding
            verticalCenter: parent.verticalCenter
        }
        spacing: root.spacing

        Repeater {
            id: repeater

            model: root.options

            AbstractButton {
                id: option

                required property var modelData
                required property int index
                readonly property bool picked: index === root.currentIndex

                width: root.optionSize
                height: root.optionSize
                padding: 5
                hoverEnabled: true
                focusPolicy: Qt.TabFocus
                onPressed: root.pick(index)

                // Moves the pick, and the focus, to the next option over.
                function step(delta) {
                    const next = repeater.itemAt(index + delta)
                    if (next) {
                        root.pick(index + delta)
                        next.forceActiveFocus(Qt.TabFocusReason)
                    }
                }

                Keys.onLeftPressed: step(-1)
                Keys.onRightPressed: step(1)

                HoverHandler {
                    cursorShape: option.picked ? Qt.ArrowCursor : Qt.PointingHandCursor
                }

                ToolTip.visible: hovered && modelData.text !== undefined
                ToolTip.delay: 500
                ToolTip.text: modelData.text ?? ""

                // Only shown when focused with the keyboard.
                background: Rectangle {
                    radius: height / 2
                    color: "transparent"
                    border.width: option.visualFocus ? 1 : 0
                    border.color: Theme.iris
                }

                // The outline and filled icons, cross-faded, so the change
                // between them is smooth.
                contentItem: Item {
                    scale: option.down && !option.picked ? 0.92 : 1

                    Behavior on scale {
                        NumberAnimation {
                            duration: Motion.fast01
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.productiveStandard
                        }
                    }

                    Image {
                        anchors.fill: parent
                        source: option.modelData.icon
                        sourceSize: Qt.size(width, height)
                        fillMode: Image.PreserveAspectFit
                        opacity: option.picked ? 0 : option.hovered ? 1 : 0.6

                        Behavior on opacity {
                            NumberAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }

                    Image {
                        anchors.fill: parent
                        source: option.modelData.icon.replace("_regular.svg", "_filled.svg")
                        sourceSize: Qt.size(width, height)
                        fillMode: Image.PreserveAspectFit
                        opacity: option.picked ? 1 : 0

                        Behavior on opacity {
                            NumberAnimation {
                                duration: Motion.fast01
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.productiveStandard
                            }
                        }
                    }
                }
            }
        }
    }
}
