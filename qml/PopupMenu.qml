import QtQuick
import QtQuick.Controls.Fusion

// Not named ContextMenu: Qt 6.9+ has a type of that name already.
Menu {
    id: root

    // Drawn inside the app's window rather than as a native popup window, so
    // the rounded corners don't depend on the platform supporting transparent
    // popup windows.
    popupType: Popup.Item

    // Keeps the items' highlights clear of the rounded corners.
    padding: 6

    // A Menu doesn't grow to fit its items by itself.
    width: {
        let widest = 0

        for (let i = 0; i < count; i++) {
            const item = itemAt(i)
            if (item && item.visible)
                widest = Math.max(widest, item.implicitWidth)
        }

        return Math.max(implicitBackgroundWidth, widest + leftPadding + rightPadding)
    }

    background: Rectangle {
        implicitWidth: 200
        radius: 12
        color: Theme.surface
        border.color: Theme.highlightMed
    }
}
