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
        // Each unit has its own singular and plural strings, rather than Qt's
        // "%n year(s)", which needs a translation file even for English.
        const units = [
            [365 * day, qsTr("1 year ago"), qsTr("%1 years ago")],
            [365 / 12 * day, qsTr("1 month ago"), qsTr("%1 months ago")],
            [7 * day, qsTr("1 week ago"), qsTr("%1 weeks ago")],
            [day, qsTr("1 day ago"), qsTr("%1 days ago")],
            [60 * 60, qsTr("1 hour ago"), qsTr("%1 hours ago")],
            [60, qsTr("1 minute ago"), qsTr("%1 minutes ago")],
        ]

        for (const [length, one, many] of units) {
            const count = Math.floor(elapsed / length)

            if (count >= 1)
                return count === 1 ? one : many.arg(count)
        }

        return qsTr("just now")
    }
}
