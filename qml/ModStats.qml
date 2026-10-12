import QtQuick
import QtQuick.Controls.Fusion

// A mod's views and likes on GameBanana, each after its icon.
Row {
    id: root

    property int views
    property int likes
    property color color: Theme.subtle
    property int pixelSize: 13

    component Stat: Row {
        property alias icon: statIcon.source
        property int count

        spacing: 3

        TintedIcon {
            id: statIcon

            anchors.verticalCenter: parent.verticalCenter
            width: 14
            height: 14
            color: root.color
        }

        Label {
            anchors.verticalCenter: parent.verticalCenter
            text: CompactNumber.format(parent.count)
            color: root.color
            font.pixelSize: root.pixelSize
        }
    }

    spacing: 10

    Stat {
        icon: "icons/eye_24_regular.svg"
        count: root.views
    }

    Stat {
        icon: "icons/heart_24_regular.svg"
        count: root.likes
    }
}
