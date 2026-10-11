import QtQuick
import QtQuick.Controls.Fusion
import QtQuick.Layouts

// The mod grid shows the category picked from the list on screen, or if none
// is, the category the list belongs to: all mods at the top level, or the
// category whose subcategories these are. Picking the picked category again
// un-picks it. A category with subcategories is picked by going back up to it
// with goTo(), e.g. from the breadcrumbs; clicking it opens it again.
Item {
    id: root

    clip: true

    // Where the list is, for Breadcrumbs: all mods, then the category of
    // each level opened, then the category picked from the list on screen, if
    // any. Each crumb's depth is for goTo(): the level whose list has that
    // category in it, or -1 for all mods. The last crumb is where the list
    // is, so it's never clicked.
    readonly property var path: {
        const crumbs = [{ name: qsTr("All Mods"), depth: -1 }]

        for (let i = 1; i < stack.depth; i++) {
            const level = stack.get(i, StackView.ForceLoad)

            if (level && level.category)
                crumbs.push({ name: level.category.name, depth: i - 1 })
        }

        const entries = stack.currentItem ? stack.currentItem.entries : []

        for (const entry of entries) {
            if (isSelected(entry)) {
                crumbs.push({ name: entry.name, depth: stack.depth - 1 })
                break
            }
        }

        return crumbs
    }

    // Whether the categories open last time have been opened again.
    property bool pathRestored: false
    readonly property var tree: categories.tree

    onTreeChanged: restorePath()

    // Goes back up to the level whose list has the crumb's category in it, and
    // picks that category, so the list and the breadcrumbs stay in step. -1
    // goes back to all mods.
    function goTo(depth) {
        if (depth < 0) {
            reset()

            return
        }

        if (depth >= stack.depth - 1)
            return // the last crumb: already there

        // The level above depth is the one opened from this category.
        const category = stack.get(depth + 1, StackView.ForceLoad).category

        stack.popToIndex(depth)
        select(category)
    }

    function reset() {
        stack.popToIndex(0)
        select(null)
    }

    // null for all mods.
    function select(category) {
        categories.selected = category
            ? { section: category.section, id: category.id }
            : { section: "Mod", id: 0 }

        savePath()
    }

    function pathKey(category) {
        return category.section + "/" + category.id
    }

    // Every change to the stack ends in select(), which saves it.
    function savePath() {
        const keys = []

        for (let i = 1; i < stack.depth; i++) {
            const level = stack.get(i, StackView.ForceLoad)

            if (level && level.category)
                keys.push(pathKey(level.category))
        }

        uiState.categoryPath = keys.join(",")
    }

    // Opens the categories the user had open last time, once the tree has
    // loaded. Go has already picked the category itself, and listed its mods.
    function restorePath() {
        if (pathRestored || categories.tree.length === 0)
            return

        pathRestored = true

        const keys = uiState.categoryPath === "" ? [] : uiState.categoryPath.split(",")
        let entries = categories.tree
        let backName = qsTr("All Mods")

        for (const key of keys) {
            const category = entries.find(entry => pathKey(entry) === key)

            // Gone from GameBanana since: start again from all mods.
            if (!category || category.children.length === 0) {
                reset()

                return
            }

            stack.push(level, { category: category, backName: backName }, StackView.Immediate)
            backName = category.name
            entries = category.children
        }

        // The picked category is either in the list now shown, or the one it
        // belongs to. If it's neither, it's gone too.
        const picked = categories.selected
        const isAllMods = picked.section === "Mod" && picked.id === 0
        const shown = entries.some(entry => isSelected(entry))
            || (stack.currentItem.category && isSelected(stack.currentItem.category))

        if (!isAllMods && !shown)
            reset()
    }

    function isSelected(category) {
        return categories.selected.section === category.section
            && categories.selected.id === category.id
    }

    function open(category) {
        const above = stack.currentItem.category

        stack.push(level, { category: category, backName: above ? above.name : qsTr("All Mods") })
        select(category)
    }

    function back() {
        stack.pop()
        select(stack.currentItem.category)
    }

    StackView {
        id: stack

        anchors.fill: parent
        initialItem: level

        // The outgoing and incoming lists move side by side, like one strip,
        // so they share Carbon's standard easing; the outgoing list also stays
        // nearby, one Back away.
        pushEnter: Transition {
            XAnimator {
                from: stack.width
                to: 0
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        pushExit: Transition {
            XAnimator {
                from: 0
                to: -stack.width
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        popEnter: Transition {
            XAnimator {
                from: -stack.width
                to: 0
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }

        popExit: Transition {
            XAnimator {
                from: 0
                to: stack.width
                duration: Motion.moderate02
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.productiveStandard
            }
        }
    }

    Component {
        id: level

        Item {
            id: levelItem

            // The category whose subcategories these are; null for the top level.
            property var category: null
            // Where Back goes: the level this one was opened from.
            property string backName
            readonly property var entries: category ? category.children : categories.tree

            ColumnLayout {
                anchors.fill: parent
                spacing: 4

                SidebarItem {
                    Layout.fillWidth: true
                    visible: levelItem.category !== null
                    text: levelItem.backName
                    icon.source: "icons/chevron_left_24_regular.svg"
                    onClicked: root.back()
                }

                BusyIndicator {
                    Layout.alignment: Qt.AlignHCenter
                    Layout.topMargin: 8
                    visible: running
                    running: levelItem.category === null && categories.loading
                }

                ColumnLayout {
                    Layout.fillWidth: true
                    Layout.margins: 6
                    spacing: 8
                    visible: levelItem.category === null && categories.error !== ""

                    ErrorLabel {
                        Layout.fillWidth: true
                        text: qsTr("Couldn’t load categories\n%1").arg(categories.error)
                        detail: categories.errorDetail
                        color: Theme.subtle
                    }

                    Button {
                        text: qsTr("Retry")
                        onClicked: retryCategoriesAction.trigger()
                    }
                }

                Item {
                    Layout.fillWidth: true
                    Layout.fillHeight: true

                    ListView {
                        id: list

                        anchors {
                            top: parent.top
                            bottom: parent.bottom
                            left: parent.left
                            right: scrollBar.visible ? scrollBar.left : parent.right
                            rightMargin: scrollBar.visible ? 2 : 0
                        }
                        model: levelItem.entries
                        spacing: 4
                        clip: true
                        boundsBehavior: Flickable.StopAtBounds
                        ScrollBar.vertical: scrollBar

                        delegate: SidebarItem {
                            required property var modelData

                            width: ListView.view.width
                            text: modelData.name
                            image: modelData.icon
                            chevron: modelData.children.length > 0
                            active: root.isSelected(modelData)
                            onClicked: {
                                if (chevron)
                                    root.open(modelData)
                                else
                                    root.select(active ? levelItem.category : modelData)
                            }
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
            }
        }
    }
}
