import QtQuick
import QtQuick.Controls.Fusion

SettingRow {
    id: root

    property alias checked: toggle.checked

    // Only for the user's flips, after checked has changed.
    signal toggled()

    interactive: true
    onRowPressed: toggle.flip()

    ToggleSwitch {
        id: toggle

        onToggled: root.toggled()
    }
}
