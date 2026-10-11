import QtQuick

// Shifts content by a hundredth of a pixel, so text never sits exactly half a
// device pixel off the pixel grid. Use it as a transform on anything that
// contains text: transform: PixelNudge {}
//
// Native text rendering snaps each corner of each letter to the pixel grid
// separately (floor(position * scale + 0.5) in Qt's text shader). On a
// fractionally scaled display, layouts often put text exactly on a half pixel
// (e.g. 162px is 202.5 device pixels at 125%), where tiny floating-point
// differences between letters round some of them up and others down, so the
// letters of a word wobble by a pixel. A hundredth of a pixel is invisible but
// makes them all round the same way.
Translate {
    x: 0.01
    y: 0.01
}
