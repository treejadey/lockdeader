pragma Singleton

import QtQuick

// Motion tokens from IBM's Carbon Design System:
// https://carbondesignsystem.com/elements/motion/overview/
//
// Carbon has two styles of motion:
//  - productive: efficient, responsive and subtle, for moments when the user is
//    focused on getting a task done (micro-interactions such as button states,
//    dropdowns and revealing additional information).
//  - expressive: lively and highly visible, reserved for occasional, important
//    moments that should catch attention (opening a new page, clicking the
//    primary action, system alerts and notifications).
//
// Easing curves come in three kinds:
//  - standard: the element is visible from the start to the end of the motion,
//    e.g. expanding tiles. Also for elements that leave the view but stay
//    nearby, ready to reappear on user action, such as a side panel.
//  - entrance: elements entering the view.
//  - exit: elements leaving the view for good, e.g. closing a modal or toast.
//
// Curves are Carbon's cubic-beziers in Qt's Easing.BezierSpline form,
// [x1, y1, x2, y2, 1, 1], to be used as:
//     easing.type: Easing.BezierSpline
//     easing.bezierCurve: Motion.productiveStandard
QtObject {
    // Durations, in milliseconds. The bigger the change in distance or size,
    // the longer the animation should take.
    readonly property int fast01: 70        // button and toggle micro-interactions
    readonly property int fast02: 110       // fade micro-interactions
    readonly property int moderate01: 150   // small expansions, short-distance movement
    readonly property int moderate02: 240   // expansions, system communication, toasts
    readonly property int slow01: 400       // large expansions, important system notifications
    readonly property int slow02: 700       // background dimming

    readonly property list<real> productiveStandard: [0.2, 0, 0.38, 0.9, 1, 1]
    readonly property list<real> productiveEntrance: [0, 0, 0.38, 0.9, 1, 1]
    readonly property list<real> productiveExit: [0.2, 0, 1, 0.9, 1, 1]

    readonly property list<real> expressiveStandard: [0.4, 0.14, 0.3, 1, 1, 1]
    readonly property list<real> expressiveEntrance: [0, 0, 0.3, 1, 1, 1]
    readonly property list<real> expressiveExit: [0.4, 0.14, 1, 1, 1, 1]
}
