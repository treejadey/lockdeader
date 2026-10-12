pragma Singleton

import QtQuick

// Counts in a few characters, like GameBanana's: 950, 1.2K, 200K, 3.4M.
QtObject {
    function format(n) {
        if (n < 1000)
            return Number(n).toLocaleString(Qt.locale(), "f", 0)

        if (n < 1000000)
            //: A count in thousands, e.g. "200K" views. %1 is the number.
            return qsTr("%1K").arg(short(n / 1000))

        //: A count in millions, e.g. "3.4M" views. %1 is the number.
        return qsTr("%1M").arg(short(n / 1000000))
    }

    // Rounded down, so a count is never shown as more than it is: one
    // decimal under 10 (unless it's 0), none from there.
    function short(value) {
        const decimals = value < 10 ? 1 : 0
        const factor = Math.pow(10, decimals)
        const rounded = Math.floor(value * factor) / factor

        return Number(rounded).toLocaleString(Qt.locale(), "f", rounded % 1 === 0 ? 0 : decimals)
    }
}
