import QtQuick
import QtQuick.Controls.Fusion

// A right-click menu in the app's style: rounded corners and roomy items.
// Fill it with PopupMenuItems. (Not named ContextMenu: Qt 6.9+ has a
// type of that name already.)
Menu {
    // Drawn inside the app's window rather than as a native popup window, so
    // the rounded corners don't depend on the platform supporting transparent
    // popup windows.
    popupType: Popup.Item
    // Keeps the items' highlights clear of the rounded corners.
    padding: 6

    background: Rectangle {
        implicitWidth: 200
        radius: 12
        color: Theme.surface
        border.color: Theme.highlightMed
    }
}
