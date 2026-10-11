# Interface guidelines

Lockdeader follows the parts of the [elementary OS Human Interface Guidelines](https://docs.elementary.io/hig) that make an app easy to understand. It doesn't try to look native to elementary or any other desktop, so these notes leave out the guidelines that are about that: GTK and Granite widgets, styling, the app's generic name, and how to draw icons. Where Lockdeader deliberately does something different, it says so.

If a design doesn't fit these guidelines, decide whether it's a real exception, and add it here if it is.

## Principles

- **Keep it small.** Every feature costs speed, clutter, bugs and upkeep. The app's main job (finding mods and installing them) should be obvious at a glance.
- **Use defaults, not settings.**
  - The app should be ready to use without being configured.
  - Adding a setting is a last resort, not a way out of making a design decision. Never ask users to make an engineering decision they can't judge.
  - Where a setting is needed, put it next to what it affects when that makes sense, as the volume control sits next to the sound player.
- **Find things out rather than asking.** For example, find Deadlock through Steam's library folders, rather than asking the user for the game's path.
- **Users shouldn't need help pages.**
  - No jargon in the normal UI: words like VPK, MD5, SHA-256 or HTTP status codes belong in logs.
  - An error says, in plain words, what went wrong and what the user can do about it.

## Launching

- The window appears straight away, with no splash screen. Network requests start after it's shown. Anything that has to wait for them shows that it's loading and is disabled until then.
- Using the app needs no setup on first launch. An empty view offers what to do next, like a welcome screen: the empty Installed page should offer a button that goes to the mods.
- If the user clears everything out (e.g. uninstalls every mod), the app is back to how it looked on first launch.
- **The app opens where the user left it:**
  - the page;
  - the section and category;
  - grid or list view;
  - the scroll position;
  - the search, where it makes sense.

  Closing and reopening should feel like the app was never closed.
- **Exception:** `RestoreDialog` still opens before the main window when the settings fail to load. Starting on defaults without the user knowing would be more surprising than a dialog.

## Always saved

- Changes apply as soon as they're made. No setting or dialog has a Save or Apply button. The settings page already works this way: `debounced` writes the change half a second later.
- A dialog that changes something (like `ModConfigDialog`) can still have an action button, because the change it makes (downloading and swapping files) is an action, not a preference.

## Closing and background work

- If the window is closed while mods are installing:
  - The window hides, the installs finish, and then the app quits.
  - If any install fails, a system notification says so.
  - If the user opens the app again before then, the hidden window comes back as it was. Starting a second copy must not happen, so the app needs a single-instance check.
- The app never minimises (to the tray or elsewhere) instead of closing.
- When background work finishes, say so with a toast if the window is focused, or a system notification if it isn't. Don't announce what the user can already see.

## Undo, not warnings

- A destructive action happens at once on screen and offers Undo, rather than asking "Are you sure?" first.
  - Uninstalling a mod removes it from the list straight away and shows a toast with Undo.
  - The files are deleted only once the toast has gone, or when the app quits.
- **Ctrl+Z** undoes the last such action while Undo is still possible.
- A confirmation dialog is only for actions that really can't be undone. Its button is the destructive one (see Dialogs).

## Feedback

- **Toasts** confirm something the user just did.
  - Phrase it as what happened ("Yamato Red was uninstalled", "Image copied").
  - A toast can have one action, such as Undo or View. A toast with an action also has a close button, and stays up longer. Every toast goes away by itself.
- **Info bars** sit in the view rather than over it, and stay until dealt with. They're for situations that need attention, e.g. GameBanana can't be reached, with Retry. Each has a severity:
  - **information**: something useful to know;
  - **question**: something the user should answer, but not urgently;
  - **warning**: something may go wrong;
  - **error**: something went wrong and needs the user's action.
- **Progress bars** each show one task, such as one mod's download. Don't combine different kinds of work into one bar.
- **Badges** count new things or things that need doing, never totals. The sidebar's count of mods still installing is fine, because it goes away.
- **System notifications** are rare. Use them only for things the user would want to know while away from the window.

## Dialogs

- A dialog is for a task that takes more than a few seconds. A simple list of actions is a menu. A quick task tied to one control is a popover, like the volume slider.
- **Text:**
  - The title or main text sums up the situation in a few words, with no full stop unless it's a question.
  - Any further text explains the details and side effects, in full sentences with full stops.
  - Someone who reads only the main text should still be able to choose.
- **Buttons** are in the bottom-right corner, as `AppDialog` has them:
  - the main action at the very end;
  - **Cancel** directly before it, when the dialog came from something the user did;
  - other alternatives before **Cancel**;
  - actions that don't close the dialog (e.g. **Open Folder**) at the start, on the left.
- **Never "OK".** Name the action: **Install**, **Restore Backup**, **Uninstall**.
- **Main action styles:** each dialog has at most one of these.
  - If the main action is safe, it's highlighted and has keyboard focus.
  - If it's destructive and can't be undone, it's red and does *not* have focus.

## Controls

- **Avoid controls that do nothing.**
  - If a control needs something first, disable it, e.g. **Install** until a file is picked. It must look disabled (`DisabledFill`).
  - If a control makes no sense in the current context, hide it, e.g. **Retry** on an install that didn't fail, or the search box's clear button while it's empty.
  - Don't hide something users need to find out the feature exists.
- **Checkboxes** pick items from a set, such as mods turned on or files to install. Clicking the label toggles them as well, and the label is a noun.
- **Switches** turn one feature on or off.
  - The label names the thing, not a verb or state: "NSFW mods", not "Show NSFW" or "Enable NSFW".
  - Don't use switches for items in a list; that's what checkboxes are for.
- **Choosing one option:** use radio buttons to show every option at once, and a drop-down list when there are too many to show. Use a mode switcher (`ContentSwitcher`) when each option fits in an icon or a word or two.
  - A mode switcher's options are all icons or all text, never a mix.
- **Icon-only buttons** always have a tooltip that says what pressing them does. If there's a shortcut, it goes in brackets: "Close (Esc)".
- **Text buttons and menu items** name an action or a place, never a status: "Uninstall", "Open on GameBanana", but not "Up to date". Text buttons need no tooltip.
- **Back buttons** say where they go: the parent category's name, or "All Categories" at the top. Not "Back"; the chevron already says that.

## Search

- Every list of things should be searchable.
- **When results appear:** local searches (installed mods, settings) update with each key press. Searches that go to GameBanana can wait for a pause in typing.
- **Case:** searches ignore case. Matching the case the user typed, only when it's mixed, is fine too.
- **Placeholder:** "Search" and what's searched, with no ellipsis: "Search Mods", "Search Installed Mods", "Search Settings".
- Typing anywhere in a list view should start a search.
- **Exception:** the search box stays centred in the top bar, rather than at its right end. Search is the main way into a mod browser, and the right side holds the view switcher and volume.

## Layout

- The sidebar is on the left, with the most-used items at the top. Avoid sections that expand inside other expanding sections; `CategoryNav` goes down a level instead.
- Toolbars put the most-used items first, and anything settings-like at the end. Leave a gap between groups.
- **Spacing:**
  - at least 12 px between controls and the window's edge;
  - 12 px between a label and its control;
  - 6 px between buttons side by side.
- **Alignment:**
  - Controls in a form line up on both edges.
  - Section headings line up on the left.
- **Window titles:**
  - The main window's title is the app's name, with no version or vendor.
  - A separate window's title names what it's for, then " — " (spaced em dash), then the app's name: "Restore Settings — Lockdeader".
  - Dialogs inside the window don't need a title bar.

## Writing

### Capitalisation

- **Title Case** (AP style) for:
  - window and dialog titles;
  - page and section titles;
  - buttons;
  - menu items;
  - tab and mode names;
  - search placeholders;
  - empty-state headings ("No Mods Found").

  For example: "Change Configuration…", "Restore Backup", "Install Selected", "Search Installed Mods".
  - Capitalise the first and last words, and every word of four letters or more.
  - Also capitalise nouns, pronouns, verbs, adjectives and adverbs.
  - Keep articles, and conjunctions and prepositions of three letters or fewer, in lower case: "Open on GameBanana", "Copy Image Link".
- **Sentence case** for:
  - labels next to controls (switches, checkboxes, setting rows);
  - descriptions and other running text;
  - tooltips;
  - toasts;
  - notification titles;
  - the text under an empty-state heading, and dialog messages.

  For example: "NSFW mods", "Select a mod to see its details".
- This departs from IBM Carbon, which uses sentence case everywhere. Carbon still sets the motion and the look of components.
- Names are written the way their owners write them: GameBanana, Deadlock, NSFW.

### Style

- **Be brief, and assume the reader is clever but not technical.** For example, "Couldn’t connect to GameBanana. Check your connection and try again", not a raw HTTP error.
- **Put the most important part first**, and don't repeat yourself.
- **Use hierarchy:** a short bold line, with an explanation under it in plain text.
- **Full stops:**
  - Single-sentence labels and tooltips have no full stop.
  - Multi-sentence text and the detail text of dialogs have them.
  - Questions always end in "?".

### Typography

| Use | Not | For |
| --- | --- | --- |
| … | ... | ellipses |
| ’ | ' | apostrophes: "Couldn’t" |
| “ ” | " " | quotes |
| – | - | ranges, without spaces: "5–12" |
| — | - | asides, without spaces: "the mod—once installed—…" |
| × | x | multiplication and sizes: "1920×1080" |
| − | - | minus |

### Ellipses

- **Use one on an action that needs more from the user before it happens**, such as opening a dialog to choose something: "Install Mod…", "Change Configuration…", "Restore Backup…" when it asks which backup.
- **Don't use one** on placeholders, or on a menu item that opens a submenu.
- **Fixed UI text must never be cut short.** Names that come from GameBanana may be:
  - mod names lose their end;
  - file names and paths lose their middle (`Text.ElideMiddle`), since their end is what tells them apart.

### Translation

- Every string the user sees is translatable:
  - in QML, `qsTr("…")`;
  - in Go, `tr("…")` (in `tr.go`) for strings that reach the UI, such as `installed.message`, with `fmt.Sprintf` for anything filled in.
- Keep the punctuation inside the translatable string.
- Never build a sentence out of pieces: use placeholders, as in `qsTr("Couldn’t load mods: %1").arg(error)`.
- Numbers, sizes and dates follow the locale where Qt can do it (`Qt.locale()`, `Locale.toString`).
- There are no translations yet, and right-to-left layouts aren't tested. Still, prefer start and end over left and right where QML offers both, so mirroring can work later.

## Icons

- Common actions use the standard Fluent icon for them, rather than a new one. Icons on buttons, in lists and over images are single-colour (`TintedIcon`). See CLAUDE.md for which icons and when.
- **The app icon** (when there is one):
  - has a distinctive shape, not a plain rounded square with a picture on it;
  - is drawn for each of 16, 24, 32, 48, 64 and 128 px rather than scaled;
  - is colourful, not grey;
  - is neither too thin nor too detailed to read when small.

## Desktop integration

Windows matters as much as Linux, so each of these needs both.

- **Linux:** the `.desktop` file:
  - `Name` is just "Lockdeader".
  - `Comment` starts with a verb, is in sentence case and has no full stop: "Install and manage Deadlock mods".
  - `Keywords` adds words people might search for, such as `Deadlock;Mods;GameBanana;`.
- **Windows:** the installer adds a Start menu shortcut named "Lockdeader", with the AppUserModelID "Lockdeader" that `notification/` uses, so notifications and the taskbar group under the app.
- **Taskbar progress (optional):** the app's taskbar entry can show install progress, through the Unity `LauncherEntry` D-Bus API on KDE and `ITaskbarList3` on Windows. Like any progress bar, it shows one task: the installs as a whole.

## Where the app doesn't follow these yet

- The app doesn't remember the page, category, view or scroll position between launches.
- There's no `.desktop` file or Windows installer yet.
