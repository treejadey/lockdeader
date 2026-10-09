pragma Singleton

import QtQuick

// Formats timestamps relative to the current time, e.g. "3 hours ago".
//
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
        const units = [
            ["year", 365 * day],
            ["month", 365 / 12 * day],
            ["week", 7 * day],
            ["day", day],
            ["hour", 60 * 60],
            ["minute", 60],
        ]
        for (const [name, length] of units) {
            const count = Math.floor(elapsed / length)
            if (count >= 1)
                return count + " " + name + (count === 1 ? "" : "s") + " ago"
        }
        return "just now"
    }
}
