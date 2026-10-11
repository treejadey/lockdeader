import QtQuick
import QtQuick.Controls.Fusion

// Says what went wrong in plain words. The error as Go wrote it shows on
// hover, for bug reports.
Label {
    id: root

    // "" for none.
    property string detail: ""

    color: Theme.love
    wrapMode: Text.Wrap

    HoverHandler {
        id: hover

        enabled: root.detail !== ""
    }

    ToolTip {
        visible: hover.hovered
        delay: 500
        width: Math.min(implicitWidth, 420)
        text: root.detail
        font.family: app.monospaceFont
    }
}
