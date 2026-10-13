import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// Hover and press feedback is Carbon's productive motion for button
// micro-interactions: fast-01 (70 ms) with standard easing.
AbstractButton {
    id: root

    // The page this button leads to is the one being shown.
    property bool active: false
    // A picture rather than an icon, e.g. a category's icon from GameBanana.
    property url image
    property bool chevron: false
    // How many things on its page are under way, e.g. installs; 0 for none.
    property int badge: 0
    // The last count, kept while the badge fades out.
    property int shownBadge: 0
    onBadgeChanged: if (root.badge > 0) root.shownBadge = root.badge

    implicitHeight: 34
    leftPadding: 8
    rightPadding: 8
    hoverEnabled: true
    // Reachable with Tab, but clicking doesn't take focus from where it was.
    focusPolicy: Qt.TabFocus

    scale: down ? 0.97 : 1

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
        radius: 6
        color: root.active ? Theme.highlightMed
             : root.down ? Theme.highlightLow
             : root.hovered ? Theme.lift(Theme.overlay, 0.25)
             : Theme.overlay
        border.width: root.visualFocus ? 1 : 0
        border.color: Theme.iris

        Behavior on color {
            ColorAnimation {
                duration: Motion.fast01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    contentItem: RowLayout {
        spacing: 8

        TintedIcon {
            Layout.preferredWidth: 20
            Layout.preferredHeight: 20
            visible: root.icon.source.toString() !== ""
            // Outline icons while inactive, filled while active: the filled
            // icon has the same name with "_filled" for "_regular".
            source: root.active
                ? root.icon.source.toString().replace("_regular.svg", "_filled.svg")
                : root.icon.source
            opacity: root.active || root.hovered ? 1 : 0.6

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        Image {
            Layout.preferredWidth: 20
            Layout.preferredHeight: 20
            visible: root.image.toString() !== ""
            source: root.image
            sourceSize: Qt.size(width, height)
            fillMode: Image.PreserveAspectFit
            asynchronous: true
            opacity: status === Image.Ready ? 1 : 0

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        Label {
            Layout.fillWidth: true
            text: root.text
            elide: Text.ElideRight
            color: root.active || root.hovered ? Theme.text : Theme.subtle
            font.weight: root.active ? Font.DemiBold : Font.Normal

            Behavior on color {
                ColorAnimation {
                    duration: Motion.fast01
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }
        }

        // A notice of something under way, so it comes in with Carbon's
        // expressive entrance over moderate-02 (system communication), and
        // leaves with the exit curve over the same time, long enough to see it go.
        Rectangle {
            id: bubble

            Layout.preferredWidth: Math.max(height, bubbleLabel.implicitWidth + 10)
            Layout.preferredHeight: 18
            visible: opacity > 0
            opacity: 0
            scale: 0.6
            radius: height / 2
            color: Theme.foam

            states: State {
                name: "shown"
                when: root.badge > 0

                PropertyChanges {
                    bubble.opacity: 1
                    bubble.scale: 1
                }
            }

            transitions: [
                Transition {
                    to: "shown"

                    NumberAnimation {
                        properties: "opacity,scale"
                        duration: Motion.moderate02
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.expressiveEntrance
                    }
                },
                Transition {
                    from: "shown"

                    NumberAnimation {
                        properties: "opacity,scale"
                        duration: Motion.moderate02
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.productiveExit
                    }
                }
            ]

            Label {
                id: bubbleLabel

                anchors.centerIn: parent
                text: root.shownBadge
                color: Theme.base
                font.pixelSize: 11
                font.weight: Font.DemiBold
                font.features: { "tnum": 1 }
            }
        }

        TintedIcon {
            Layout.preferredWidth: 16
            Layout.preferredHeight: 16
            visible: root.chevron
            source: "icons/chevron_right_24_regular.svg"
            opacity: root.hovered ? 1 : 0.6

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
