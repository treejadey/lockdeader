import QtQuick
import QtQuick.Controls.Fusion

Item {
    id: root

    // {title, shown} objects, as SettingsPage.sectionList.
    property var sections: []
    // -1 means none.
    property int currentIndex: -1

    signal sectionClicked(int index)

    ListView {
        id: list

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            right: scrollBar.visible ? scrollBar.left : parent.right
            rightMargin: scrollBar.visible ? 2 : 0
        }
        // The shown sections, with their indexes among all of them.
        model: root.sections
            .map((section, index) => ({ title: section.title, shown: section.shown, index: index }))
            .filter(section => section.shown)

        spacing: 4
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        ScrollBar.vertical: scrollBar

        delegate: SidebarItem {
            required property var modelData

            width: ListView.view.width
            text: modelData.title
            active: modelData.index === root.currentIndex
            onClicked: root.sectionClicked(modelData.index)
        }
    }

    SmoothWheel {
        id: wheel

        anchors.fill: list
        flickable: list
    }

    // Only takes up room when the list doesn't fit.
    ScrollBar {
        id: scrollBar

        anchors {
            top: parent.top
            bottom: parent.bottom
            right: parent.right
        }
        visible: list.contentHeight > list.height
        policy: ScrollBar.AlwaysOn
        onPressedChanged: if (pressed) wheel.stop()
    }
}
