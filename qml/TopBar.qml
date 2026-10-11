import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    // As Breadcrumbs crumbs; empty for none.
    property var crumbs: []
    property bool showVolume: false
    // Not shown along with the volume, which takes its place.
    property bool showViews: false
    // 0 for the grid, 1 for the list.
    property alias viewIndex: views.currentIndex
    property alias searchText: field.text
    property string searchPlaceholder

    // The search field never gets narrower than this...
    readonly property int minimumSearchWidth: 220
    // ...as long as the bar is at least this wide.
    readonly property int minimumWidth: minimumSearchWidth + 2 * (volume.width + 24)

    signal crumbClicked(int depth)
    signal searchEdited()
    signal searchAccepted()

    implicitHeight: 56

    Breadcrumbs {
        anchors {
            left: parent.left
            leftMargin: 16
            right: field.left
            rightMargin: 16
            verticalCenter: parent.verticalCenter
        }
        crumbs: root.crumbs
        onCrumbClicked: (depth) => root.crumbClicked(depth)
    }

    SearchBox {
        id: field

        anchors.centerIn: parent
        placeholderText: root.searchPlaceholder
        // Never so wide that it runs into the volume at either side. Kept
        // the same while the volume is hidden, so the field never jumps.
        width: Math.max(root.minimumSearchWidth, Math.min(300, parent.width - 2 * (volume.width + 24)))
        onTextEdited: root.searchEdited()
        onAccepted: root.searchAccepted()
    }

    VolumeControl {
        id: volume

        anchors {
            right: parent.right
            rightMargin: 12
            verticalCenter: parent.verticalCenter
        }
        opacity: root.showVolume ? 1 : 0
        visible: opacity > 0

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    ContentSwitcher {
        id: views

        anchors {
            right: parent.right
            rightMargin: 12
            verticalCenter: parent.verticalCenter
        }
        options: [
            { icon: "icons/grid_24_regular.svg", text: "Grid view" },
            { icon: "icons/text_bullet_list_ltr_24_regular.svg", text: "List view" },
        ]

        opacity: root.showViews && !root.showVolume ? 1 : 0
        visible: opacity > 0

        Behavior on opacity {
            NumberAnimation {
                duration: Motion.fast02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }
}
