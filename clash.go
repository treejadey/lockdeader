package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/merge"
)

// Only one of two mods that change the same files can be on (see
// deploy.Clashes): turning one on turns off the others, with a message that
// offers Undo. A mod that's off for clashing with ones that are on says
// which in its "clashes" role.

// pendingToggle is mods turned on and off by turnOffClashing or
// ResolveClashes, which Undo turns back.
type pendingToggle struct {
	messageID int
	on, off   []string // keys
}

// vpkPaths reads a stored VPK's file list once per run: the store names
// VPKs by their SHA-256, so what a file holds never changes. One that can't
// be read counts as having no files, so it clashes with nothing.
func (m *installedMods) vpkPaths(sha256 string) deploy.PathSet {
	if set, ok := m.paths[sha256]; ok {
		return set
	}

	set, err := deploy.ReadPaths(m.store.Path(sha256))
	if err != nil {
		slog.Warn("couldn't read which files a VPK has, so it can't be checked for clashes", "err", err)

		set = deploy.PathSet{}
	}

	m.paths[sha256] = set

	return set
}

// modPaths is the files of the VPKs the mod uses.
func (m *installedMods) modPaths(row *installedRow) deploy.PathSet {
	var sets []deploy.PathSet

	for _, f := range row.mod.Files {
		for _, v := range f.VPKs {
			if v.Enabled {
				sets = append(sets, m.vpkPaths(v.SHA256))
			}
		}
	}

	if len(sets) == 1 {
		return sets[0]
	}

	all := deploy.PathSet{}

	for _, set := range sets {
		for p, c := range set {
			all[p] = c
		}
	}

	return all
}

// installedRows is the rows of the installed mods, in load order.
func (m *installedMods) installedRows() []*installedRow {
	var rows []*installedRow

	for _, row := range m.rows {
		if row.mod != nil {
			rows = append(rows, row)
		}
	}

	slices.SortFunc(rows, func(a, b *installedRow) int {
		if a.mod.LoadOrder != b.mod.LoadOrder {
			return a.mod.LoadOrder - b.mod.LoadOrder
		}

		return int(a.mod.ID - b.mod.ID)
	})

	return rows
}

// clashes returns the installed rows, in load order, and for each the
// indexes of those it clashes with.
func (m *installedMods) clashes() ([]*installedRow, [][]int) {
	rows := m.installedRows()
	sets := make([]deploy.PathSet, len(rows))

	for i, row := range rows {
		sets[i] = m.modPaths(row)
	}

	mergeable := func(path string, a, b int) bool {
		return m.merges(path, rows[a], rows[b], sets[a], sets[b])
	}

	return rows, deploy.Clashes(sets, mergeable)
}

// merges says whether two mods' different copies of a sound event table
// merge without a conflict (see merge.SoundEvents), so they don't clash over
// it. Without the game's own copy to merge over, or with a copy that can't
// be read, they do. The answer is kept for the game version and the mods'
// VPKs, which decide it.
func (m *installedMods) merges(path string, a, b *installedRow, setA, setB deploy.PathSet) bool {
	if !strings.HasSuffix(path, ".vsndevts_c") {
		return false
	}

	base := m.gameBase()
	if base == nil {
		return false
	}

	key := strings.Join([]string{base.Stamp(), path, vpkKey(a), vpkKey(b)}, "|")

	if ok, seen := m.mergeable[key]; seen {
		return ok
	}

	ok := m.mergeCleanly(base, path, []*installedRow{a, b}, []deploy.PathSet{setA, setB})
	m.mergeable[key] = ok

	return ok
}

func (m *installedMods) mergeCleanly(base *deploy.Base, path string, rows []*installedRow, sets []deploy.PathSet) bool {
	game, _, err := base.File(path)
	if err != nil {
		slog.Warn("couldn't read the game's copy of a table, so mods that change it clash", "path", path, "err", err)

		return false
	}

	copies := make([]merge.Copy, len(rows))

	for i, row := range rows {
		data, err := m.modFile(row, path)
		if err != nil {
			slog.Warn("couldn't read a mod's copy of a table, so it clashes with others that change it",
				"mod", row.key, "path", path, "err", err)

			return false
		}

		copies[i] = merge.Copy{Data: data, Ships: func(p string) bool { _, ok := sets[i][p]; return ok }}
	}

	conflicts, err := merge.SoundEventConflicts(game, copies)
	if err != nil {
		slog.Warn("couldn't merge mods' copies of a table, so they clash", "path", path, "err", err)

		return false
	}

	return len(conflicts) == 0
}

// gameBase is the base game, read again after an update, or nil without a
// game folder.
func (m *installedMods) gameBase() *deploy.Base {
	if m.gameFolder == nil {
		return nil
	}

	folder := m.gameFolder()
	if folder == "" {
		return nil
	}

	if m.base != nil && m.base.Current(folder) {
		return m.base
	}

	base, err := deploy.ReadBase(folder)
	if err != nil {
		slog.Warn("couldn't read the base game, so mods that change the same tables clash", "err", err)
	}

	m.base = base

	return base
}

// modFile reads a file from the first of a mod's VPKs that has it.
func (m *installedMods) modFile(row *installedRow, path string) ([]byte, error) {
	var mod deploy.Mod

	for _, f := range row.mod.Files {
		for _, v := range f.VPKs {
			if v.Enabled {
				mod.Files = append(mod.Files, deploy.File{SHA256: v.SHA256, Path: m.store.Path(v.SHA256), Size: v.Size})
			}
		}
	}

	return mod.ReadFile(path)
}

// vpkKey names what a mod is made of: its VPKs that are used.
func vpkKey(row *installedRow) string {
	var shas []string

	for _, f := range row.mod.Files {
		for _, v := range f.VPKs {
			if v.Enabled {
				shas = append(shas, v.SHA256)
			}
		}
	}

	return strings.Join(shas, ",")
}

// updateClashes fills in every row's "clashes" role: for a mod that's off,
// the names of the mods that are on that it clashes with, which keep it
// off; otherwise "". A mod that's on is fine, and so is one that only
// clashes with mods that are off, so they show nothing.
func (m *installedMods) updateClashes() {
	rows, clashes := m.clashes()
	names := make(map[*installedRow]string, len(rows))

	for i, row := range rows {
		if row.mod.Enabled {
			continue
		}

		var list []string

		for _, j := range clashes[i] {
			if rows[j].mod.Enabled {
				list = append(list, rows[j].mod.Name)
			}
		}

		names[row] = strings.Join(list, ", ")
	}

	for _, row := range m.rows {
		row.item.SetData(qt.NewQVariant14(names[row]), roleInstalledClashes)
	}
}

// settle is for after row's mod was installed, turned on or changed: if
// it's on, the mods it clashes with are turned off, and which mods clash is
// worked out again.
func (m *installedMods) settle(row *installedRow) {
	if row.mod != nil && row.mod.Enabled {
		m.turnOffClashing(row)
	}

	m.updateClashes()
}

func (m *installedMods) turnOffClashing(row *installedRow) {
	rows, clashes := m.clashes()

	index := slices.Index(rows, row)
	if index < 0 {
		return
	}

	var off []*installedRow

	for _, j := range clashes[index] {
		if rows[j].mod.Enabled && m.writeEnabled(rows[j], false) {
			off = append(off, rows[j])
		}
	}

	if len(off) == 0 {
		return
	}

	slog.Info("turned off mods that change the same files", "mod", row.key, "off", len(off))

	if len(off) == 1 {
		m.notifyUndoable(fmt.Sprintf(tr("Turned off %s, which changes the same files"), off[0].mod.Name))
	} else {
		m.notifyUndoable(trN("Turned off %n mod that changes the same files",
			"Turned off %n mods that change the same files", len(off)))
	}

	m.toggled = &pendingToggle{messageID: m.messages, on: []string{row.key}, off: keys(off)}
}

// SetMerged fills in every row's "mergedWith" role from the last apply:
// for a mod the merge of shared tables left some of its changes out, the
// names of the mods it's merged with; otherwise "". leftOut is by key.
func (m *installedMods) SetMerged(leftOut map[string][]string) {
	for key, row := range m.rows {
		var names []string

		for _, other := range leftOut[key] {
			if o := m.rows[other]; o != nil && o.mod != nil {
				names = append(names, o.mod.Name)
			}
		}

		row.item.SetData(qt.NewQVariant14(strings.Join(names, ", ")), roleInstalledMergedWith)
	}
}

// ResolveClashes turns off the mods that clash with one that's on and wins
// over them in the load order. Mods installed before Lockdeader kept clashing
// mods apart can be on together; call it once QML is listening for messages.
func (m *installedMods) ResolveClashes() {
	rows, clashes := m.clashes()
	kept := make([]bool, len(rows))

	var off []*installedRow

	for i, row := range rows {
		if !row.mod.Enabled {
			continue
		}

		// Lower in the load order wins, and rows are in load order, so any
		// mod that beats this one has been kept or turned off already.
		beaten := slices.ContainsFunc(clashes[i], func(j int) bool { return kept[j] })

		if !beaten {
			kept[i] = true
		} else if m.writeEnabled(row, false) {
			off = append(off, row)
		}
	}

	if len(off) == 0 {
		return
	}

	slog.Info("turned off mods that clash with others on startup", "off", len(off))

	if len(off) == 1 {
		m.notifyUndoable(fmt.Sprintf(tr("Turned off %s, which changes the same files as another mod"), off[0].mod.Name))
	} else {
		m.notifyUndoable(trN("Turned off %n mod that changes the same files as others",
			"Turned off %n mods that change the same files as others", len(off)))
	}

	m.toggled = &pendingToggle{messageID: m.messages, off: keys(off)}
	m.changed()
}

func (m *installedMods) undoToggle(p *pendingToggle) {
	for _, key := range p.on {
		if row := m.rows[key]; row != nil && row.mod != nil {
			m.writeEnabled(row, false)
		}
	}

	for _, key := range p.off {
		if row := m.rows[key]; row != nil && row.mod != nil {
			m.writeEnabled(row, true)
		}
	}

	slog.Info("undid turning off mods that change the same files", "off", len(p.off))

	m.changed()
}

// writeEnabled turns the mod on or off, in the database and on its row, and
// says whether that worked.
func (m *installedMods) writeEnabled(row *installedRow, enabled bool) bool {
	if err := m.db.SetModEnabled(context.Background(), row.mod.ID, enabled); err != nil {
		slog.Error("failed to turn a mod on or off", "mod", row.key, "err", err)

		return false
	}

	row.mod.Enabled = enabled
	row.item.SetData(qt.NewQVariant8(enabled), roleInstalledEnabled)

	return true
}

func keys(rows []*installedRow) []string {
	keys := make([]string, len(rows))

	for i, row := range rows {
		keys[i] = row.key
	}

	return keys
}
