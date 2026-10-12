pragma Singleton

import QtQuick

// Pass `now` into format() from a binding, e.g.
//     text: RelativeTime.format(timestamp, RelativeTime.now)
// so the text updates as `now` ticks over, every minute.
QtObject {
    id: root

    // Milliseconds since the epoch, like Date.now().
    property double now: Date.now()

    property Timer ticker: Timer {
        interval: 60 * 1000
        running: true
        repeat: true
        onTriggered: root.now = Date.now()
    }

    // seconds is a Unix timestamp in seconds; 0 means unknown and gives "".
    function format(seconds, now) {
        if (!seconds)
            return ""

        const day = 24 * 60 * 60
        const elapsed = Math.max(0, now / 1000 - seconds)

        // A month is a twelfth of a year, so 11 months never rounds to 12.
        // Each qsTr is written out in full, for the translations tool to
        // find; the English plural takes the disambiguation's place.
        const units = [
            [365 * day, (n) => qsTr("%n year ago", "%n years ago", n)],
            [365 / 12 * day, (n) => qsTr("%n month ago", "%n months ago", n)],
            [7 * day, (n) => qsTr("%n week ago", "%n weeks ago", n)],
            [day, (n) => qsTr("%n day ago", "%n days ago", n)],
            [60 * 60, (n) => qsTr("%n hour ago", "%n hours ago", n)],
            [60, (n) => qsTr("%n minute ago", "%n minutes ago", n)],
        ]

        for (const [length, text] of units) {
            const count = Math.floor(elapsed / length)

            if (count >= 1)
                return text(count)
        }

        return qsTr("just now")
    }
}
