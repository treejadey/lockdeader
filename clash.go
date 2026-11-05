package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"github.com/treejadey/lockdeader/deploy"
)

// Only one of two mods that change the same files can be on (see
// deploy.Clashes): turning one on turns off the others, with a message that
// offers Undo. Every installed mod that clashes with another, on or off,
// says which in its "clashes" role.

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

	return rows, deploy.Clashes(sets)
}

// updateClashes fills in every row's "clashes" role: the names of the
// installed mods it clashes with, or "".
func (m *installedMods) updateClashes() {
	rows, clashes := m.clashes()
	names := make(map[*installedRow]string, len(rows))

	for i, row := range rows {
		var list []string

		for _, j := range clashes[i] {
			list = append(list, rows[j].mod.Name)
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
