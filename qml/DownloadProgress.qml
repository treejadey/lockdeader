import QtQuick
import QtQuick.Effects

// A download's progress bar, in the style of elementary OS's: a rounded
// track, filled from the left with rounded ends, and a few brighter slanted
// bands (the shine) sweeping through the filled part together, then pausing
// before the next sweep.
// (Not named ProgressBar: Qt Quick Controls has a type of that name already.)
//
// How the shine moves follows how fast the download is going (pace):
//  - normal: sweeps through, then pauses;
//  - slow: sweeps through more slowly, then pauses as long as at normal;
//  - fast: sweeps through quicker, and never pauses.
// A change of pace takes effect from the next sweep, so the shine never jumps.
//
// Motion: each sweep accelerates through the bar (Carbon's productive exit
// curve, which is an ease-in), as the shine leaves the bar at its end. The
// sweep and pause times make a loop rather than a transition, so they aren't
// Carbon's durations. The fill grows to each new value over moderate-01, so
// progress reported a few times a second still moves smoothly.
Item {
    id: root

    // How much is done, from 0 to 1.
    property real value: 0
    // "slow", "normal" or "fast".
    property string pace: "normal"
    // Whether the shine sweeps, e.g. only while something is happening.
    property bool running: true
    property color trackColor: Theme.overlay

    // Sweep and pause times, in milliseconds, for each pace.
    readonly property var timings: ({
        slow: { sweep: 3200, pause: 1000 },
        normal: { sweep: 2100, pause: 1000 },
        fast: { sweep: 1400, pause: 0 },
    })
    // The shine's bands: how many, how wide each is and the gap between
    // them, both as parts of the bar's width.
    readonly property int bands: 3
    readonly property real bandWidth: 0.26
    readonly property real bandGap: 0.14
    readonly property var timing: timings[pace] ?? timings.normal
    // How far the shine's bottom edge is ahead of its top, per pixel of height.
    readonly property real slant: 0.75

    implicitWidth: 120
    implicitHeight: 6

    Rectangle {
        anchors.fill: parent
        radius: height / 2
        color: root.trackColor
    }

    Item {
        id: fill

        // Never narrower than it is tall, so its ends stay round.
        width: root.value > 0 ? Math.max(root.height, Math.min(1, root.value) * root.width) : 0
        height: root.height
        visible: width > 0

        Behavior on width {
            NumberAnimation {
                duration: Motion.moderate01
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        // The fill and its shine, cut to the fill's rounded shape.
        layer.enabled: true
        layer.effect: MultiEffect {
            maskEnabled: true
            maskSource: fillShape
            // Softens the mask's edge, so the ends are antialiased.
            maskThresholdMin: 0.5
            maskSpreadAtMin: 1.0
        }

        // Between Rosé Pine's pine and foam, so the foam shine stands out.
        Rectangle {
            anchors.fill: parent
            color: Qt.rgba(Theme.pine.r + (Theme.foam.r - Theme.pine.r) * 0.35,
                           Theme.pine.g + (Theme.foam.g - Theme.pine.g) * 0.35,
                           Theme.pine.b + (Theme.foam.b - Theme.pine.b) * 0.35, 1)
        }

        // The bands, one after another, moving as one.
        Item {
            id: shine

            // How far through its sweep it is, from 0 (all of it just before
            // the start of the fill) to 1 (all of it just past its end).
            property real travel: 0
            readonly property real bandWidth: Math.max(2 * root.height, root.width * root.bandWidth)
            readonly property real gap: Math.max(root.height, root.width * root.bandGap)
            readonly property real lean: bandHeight * root.slant
            // A little in from the top and bottom of the bar.
            readonly property real inset: Math.max(1, Math.round(root.height * 0.2))
            readonly property real bandHeight: root.height - 2 * inset

            width: root.bands * bandWidth + (root.bands - 1) * gap + lean
            height: root.height
            x: -width + travel * (fill.width + width)
            visible: travel > 0 && travel < 1

            Repeater {
                model: root.bands

                Rectangle {
                    required property int index

                    x: index * (shine.bandWidth + shine.gap)
                    y: shine.inset
                    width: shine.bandWidth
                    height: shine.bandHeight
                    color: Theme.foam
                    // Slants it: the bottom edge is ahead of the top, as if moving.
                    transform: Matrix4x4 {
                        matrix: Qt.matrix4x4(1, root.slant, 0, 0,
                                             0, 1, 0, 0,
                                             0, 0, 1, 0,
                                             0, 0, 0, 1)
                    }
                }
            }
        }
    }

    Item {
        id: fillShape

        width: fill.width
        height: fill.height
        visible: false
        layer.enabled: true

        Rectangle {
            anchors.fill: parent
            radius: height / 2
            color: "black"
        }
    }

    // One sweep and its pause. It's started again when it ends, rather than
    // looping, so each sweep picks up the timing for the pace at the time.
    SequentialAnimation {
        id: cycle

        NumberAnimation {
            target: shine
            property: "travel"
            from: 0
            to: 1
            duration: root.timing.sweep
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.productiveExit
        }

        PauseAnimation {
            duration: root.timing.pause
        }

        onFinished: Qt.callLater(root.updateShine)
    }

    // Starts or stops the shine as it's wanted or not; only when it's shown.
    function updateShine() {
        const wanted = running && visible && fill.visible
        if (wanted && !cycle.running) {
            cycle.start()
        } else if (!wanted && cycle.running) {
            cycle.stop()
            shine.travel = 0
        }
    }

    onRunningChanged: updateShine()
    onVisibleChanged: updateShine()
    Component.onCompleted: updateShine()

    Connections {
        target: fill
        function onVisibleChanged() { root.updateShine() }
    }
}
