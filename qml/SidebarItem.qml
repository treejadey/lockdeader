import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// A navigation button in the sidebar: a label, optionally with an icon or an
// image before it, and a chevron (for "leads to more options") after it. A
// count after the label, if it has one, says how many things on its page are
// under way, e.g. installs.
//
// Hover and press feedback is Carbon's productive motion for button
// micro-interactions: fast-01 (70 ms) with standard easing. Hovering lightens
// the background and brightens the icon and label; pressing darkens the
// background and shrinks the button slightly, as if pushed in.
AbstractButton {
    id: root

    // The page this button leads to is the one being shown.
    property bool active: false
    // A picture shown before the label, e.g. a category's icon from GameBanana.
    property url image
    // Shows a chevron at the right edge: clicking opens more options.
    property bool chevron: false
    // The count shown after the label; 0 for none.
    property int badge: 0
    // The last count, kept while the badge fades out.
    property int shownBadge: 0
    onBadgeChanged: if (badge > 0) shownBadge = badge

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
             : root.hovered ? Qt.lighter(Theme.overlay, 1.25)
             : Theme.overlay
        // Only shown when focused with the keyboard.
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

        Image {
            Layout.preferredWidth: 20
            Layout.preferredHeight: 20
            visible: root.icon.source.toString() !== ""
            // Outline icons while inactive, filled while active: the filled
            // icon has the same name with "_filled" for "_regular".
            source: root.active
                ? root.icon.source.toString().replace("_regular.svg", "_filled.svg")
                : root.icon.source
            sourceSize: Qt.size(width, height)
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
            // Fades in once loaded, rather than popping in.
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

        // Fades and grows in as it appears, and back out as it goes.
        Rectangle {
            Layout.preferredWidth: Math.max(height, badgeLabel.implicitWidth + 10)
            Layout.preferredHeight: 18
            visible: opacity > 0
            opacity: root.badge > 0 ? 1 : 0
            scale: root.badge > 0 ? 1 : 0.6
            radius: height / 2
            color: Theme.foam

            Behavior on opacity {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            Behavior on scale {
                NumberAnimation {
                    duration: Motion.fast02
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.productiveStandard
                }
            }

            Label {
                id: badgeLabel

                anchors.centerIn: parent
                text: root.shownBadge
                color: Theme.base
                font.pixelSize: 11
                font.weight: Font.DemiBold
                font.features: { "tnum": 1 }
            }
        }

        Image {
            Layout.preferredWidth: 16
            Layout.preferredHeight: 16
            visible: root.chevron
            source: "icons/chevron_right_24_regular.svg"
            sourceSize: Qt.size(width, height)
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
