import QtQuick
import QtQuick.Controls.Fusion

// How a mod's install is going: a progress bar, and how far along it is
// ("33%"), or that it's waiting its turn or installing what it downloaded.
// The label goes under the bar, at its right, or beside it. A failed install
// says so instead, in red, with why when hovered.
Item {
    id: root

    // The mod's status, as in the installed mods model: "queued",
    // "downloading", "extracting" or "failed".
    property string status
    // How much has downloaded, from 0 to 1.
    property real progress: 0
    // How fast it's downloading: "slow", "normal" or "fast".
    property string pace: "normal"
    // Why it failed.
    property string error
    // Puts the label beside the bar rather than under it.
    property bool labelBeside: false
    property alias trackColor: bar.trackColor

    readonly property bool failed: status === "failed"

    implicitWidth: 160
    // The same whatever the status, so the label stays put as it changes.
    implicitHeight: labelBeside ? Math.max(bar.implicitHeight, label.implicitHeight)
                                : bar.implicitHeight + 4 + label.implicitHeight

    DownloadProgress {
        id: bar

        anchors {
            left: parent.left
            right: root.labelBeside ? label.left : parent.right
            rightMargin: root.labelBeside ? 8 : 0
            top: root.labelBeside ? undefined : parent.top
            verticalCenter: root.labelBeside ? parent.verticalCenter : undefined
        }
        visible: !root.failed
        value: root.status === "extracting" ? 1 : root.progress
        pace: root.status === "extracting" ? "normal" : root.pace
        running: root.status === "downloading" || root.status === "extracting"
    }

    Label {
        id: label

        anchors {
            right: parent.right
            bottom: root.labelBeside ? undefined : parent.bottom
            verticalCenter: root.labelBeside ? parent.verticalCenter : undefined
        }
        // Room for "100%", so the bar's length doesn't change as it counts up.
        width: Math.max(implicitWidth, percentMetrics.advanceWidth("100%"))
        horizontalAlignment: Text.AlignRight
        text: root.failed ? "Couldn't install"
            : root.status === "queued" ? "Waiting"
            : root.status === "extracting" ? "Installing"
            : Math.floor(Math.min(1, root.progress) * 100) + "%"
        color: root.failed ? Theme.love : Theme.subtle
        font.pixelSize: 12
        font.weight: Font.Medium
        // Figures of the same width, so the percentage doesn't shift as it counts.
        font.features: { "tnum": 1 }

        FontMetrics {
            id: percentMetrics
            font: label.font
        }

        HoverHandler {
            id: errorHover
            enabled: root.failed
        }

        ToolTip {
            visible: errorHover.hovered && root.error !== ""
            delay: 500
            text: root.error
            width: Math.min(implicitWidth, 360)
        }
    }
}
