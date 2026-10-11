import QtQuick
import QtQuick.Controls.Fusion

// A setting that's on or off, with a ToggleSwitch. Pressing anywhere on the
// row flips it.
SettingRow {
    id: root

    property alias checked: toggle.checked

    // Emitted when the user flips the switch, after checked has changed.
    signal toggled()

    interactive: true
    onRowPressed: toggle.flip()

    ToggleSwitch {
        id: toggle
        onToggled: root.toggled()
    }
}
