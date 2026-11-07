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
    // The orders the list can be in, as DropDown options; empty for none.
    property var sorts: []
    property var sortId

    // The search field never gets narrower than this...
    readonly property int minimumSearchWidth: 220
    // Kept clear on the left for the breadcrumbs, and on the right for the
    // controls there, whichever of them are shown, so nothing moves when
    // they're swapped.
    readonly property real leftReserve: volume.width + 24
    readonly property real rightReserve: 12 + Math.max(volume.width, views.width)
        + (sort.visible ? sort.width + 8 : 0) + 24
    // ...as long as the bar is at least this wide.
    readonly property int minimumWidth: Math.max(minimumSearchWidth + 2 * leftReserve,
        leftReserve + minimumSearchWidth + rightReserve)

    signal crumbClicked(int depth)
    signal searchEdited()
    signal searchAccepted()
    signal sortPicked(var id)

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

        // Centred, unless that would run it into the controls on the right.
        x: Math.round(Math.max(root.leftReserve,
            Math.min((parent.width - width) / 2, parent.width - root.rightReserve - width)))
        anchors.verticalCenter: parent.verticalCenter
        placeholderText: root.searchPlaceholder
        // Never so wide that it runs into the volume at either side. Kept
        // the same while the volume is hidden, so the field never jumps.
        width: Math.max(root.minimumSearchWidth, Math.min(300, parent.width - 2 * (volume.width + 24)))
        onTextEdited: root.searchEdited()
        onAccepted: root.searchAccepted()
    }

    DropDown {
        id: sort

        anchors {
            right: root.showVolume ? volume.left : views.left
            rightMargin: 8
            verticalCenter: parent.verticalCenter
        }
        visible: root.sorts.length > 0
        options: root.sorts
        currentId: root.sortId
        onPicked: (id) => root.sortPicked(id)
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
            { icon: "icons/grid_24_regular.svg", text: qsTr("Grid view") },
            { icon: "icons/text_bullet_list_ltr_24_regular.svg", text: qsTr("List view") },
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
