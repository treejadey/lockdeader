package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
)

// A mod's configuration is which of its options it uses. Its options are
// the files it has on GameBanana, often versions of it (e.g. "flag only" or
// "music only"), and of the archives it has installed that have more than
// one VPK in them, each VPK. Picking an option that isn't installed
// downloads it; unpicking every option of an installed archive removes it.

// option is one of a mod's options: an installed VPK, or a file on
// GameBanana that isn't installed.
type option struct {
	id     string // "vpk:<id in the database>" or "file:<GameBanana file id>"
	name   string // the archive's name
	detail string // the VPK's path in the archive, or the file's description
	size   int64
	added  time.Time // when the archive was uploaded
	picked bool      // the mod uses it
	// For an option that isn't installed: the file to download.
	file *banana.File
}

// options returns an installed mod's options, in the order GameBanana lists
// its files, then any installed files that are no longer on GameBanana.
func (m *installedMods) options(row *installedRow) []option {
	if row.mod == nil {
		return nil
	}
	available := m.available[row.key]
	descriptions := make(map[int]string, len(available))
	for _, f := range available {
		descriptions[f.ID] = f.Description
	}

	installedOptions := func(f appdb.ModFile) []option {
		var opts []option
		for _, v := range f.VPKs {
			o := option{
				id: fmt.Sprintf("vpk:%d", v.ID), name: f.Name, detail: descriptions[f.GameBananaID],
				size: f.Size, added: f.UploadedAt, picked: v.Enabled,
			}
			if len(f.VPKs) > 1 {
				o.detail, o.size = v.Path, v.Size
			}
			opts = append(opts, o)
		}
		return opts
	}

	installed := make(map[int]appdb.ModFile, len(row.mod.Files))
	for _, f := range row.mod.Files {
		installed[f.GameBananaID] = f
	}
	var opts []option
	for _, f := range available {
		if inst, ok := installed[f.ID]; ok {
			opts = append(opts, installedOptions(inst)...)
			delete(installed, f.ID)
			continue
		}
		opts = append(opts, option{
			id: fmt.Sprintf("file:%d", f.ID), name: f.Name, detail: f.Description,
			size: f.Size, added: time.Unix(f.DateAdded, 0), file: &f,
		})
	}
	for _, f := range row.mod.Files {
		if _, ok := installed[f.GameBananaID]; ok {
			opts = append(opts, installedOptions(f)...)
		}
	}
	return opts
}

// configure fills in "configuration" for the configuration dialog, with an
// installed mod's options:
//
//	key:     the mod's key
//	name:    its name
//	loading: its files on GameBanana are being looked up; until they're
//	         known, only its installed options are listed
//	error:   why looking them up failed, if it did
//	options: {id, name, detail, size, added, picked, installed} objects:
//	         size is formatted, e.g. "450 KB", and added is when the archive
//	         was uploaded, in Unix seconds
func (m *installedMods) configure(key string) {
	row := m.rows[key]
	if row == nil || row.mod == nil {
		return
	}
	m.configuring = key
	m.publishConfiguration(row, "")
	m.lookUp(row)
}

func (m *installedMods) publishConfiguration(row *installedRow, lookupErr string) {
	var list []qt.QVariant
	for _, o := range m.options(row) {
		list = append(list, *qt.NewQVariant20(map[string]qt.QVariant{
			"id":        *qt.NewQVariant14(o.id),
			"name":      *qt.NewQVariant14(o.name),
			"detail":    *qt.NewQVariant14(o.detail),
			"size":      *qt.NewQVariant14(formatSize(o.size)),
			"added":     *qt.NewQVariant6(o.added.Unix()),
			"picked":    *qt.NewQVariant8(o.picked),
			"installed": *qt.NewQVariant8(o.file == nil),
		}))
	}
	m.Insert("configuration", qt.NewQVariant20(map[string]qt.QVariant{
		"key":     *qt.NewQVariant14(row.key),
		"name":    *qt.NewQVariant14(row.mod.Name),
		"loading": *qt.NewQVariant8(m.lookingUp[row.key]),
		"error":   *qt.NewQVariant14(lookupErr),
		"options": *qt.NewQVariant43(list),
	}))
}

// lookUpAll looks up which files the installed mods have on GameBanana, for
// those it isn't known for yet, a couple at a time.
func (m *installedMods) lookUpAll() {
	for _, row := range m.rows {
		m.lookUp(row)
	}
}

// lookUp looks up which files an installed mod has on GameBanana, unless
// that's known already, or being looked up.
func (m *installedMods) lookUp(row *installedRow) {
	if row.mod == nil || m.available[row.key] != nil || m.lookingUp[row.key] {
		return
	}
	key, section, id := row.key, banana.Section(row.mod.Section), row.mod.GameBananaID
	m.lookingUp[key] = true
	go func() {
		m.lookupSlots <- struct{}{}
		p, err := m.client.GetModProfile(context.Background(), section, id)
		<-m.lookupSlots

		mainthread.Start(func() {
			delete(m.lookingUp, key)
			errText := ""
			if err != nil {
				slog.Error("failed to look up a mod's files", "mod", key, "err", err)
				errText = err.Error()
			} else {
				// Not nil, even with no files, so it isn't looked up again.
				m.available[key] = append([]banana.File{}, p.Files...)
			}
			row := m.rows[key]
			if row == nil || row.mod == nil {
				return // uninstalled in the meantime
			}
			if row.job == nil {
				m.show(row)
			}
			if m.configuring == key {
				m.publishConfiguration(row, errText)
			}
		})
	}()
}

// configChange is what's left to do to change a mod's configuration once
// the files it needs are installed.
type configChange struct {
	remove []int64 // the installed files to remove, by id in the database
	enable []int64 // the VPKs to use, by id in the database; the rest aren't
}

// Configure changes which of an installed mod's options it uses: those with
// the given ids (see option), and none of its others. At least one has to be
// picked. Options that aren't installed are downloaded first, and nothing
// else changes until they are, so a failed download leaves the mod as it was.
func (m *installedMods) Configure(key string, ids []string) {
	row := m.rows[key]
	if row == nil || row.mod == nil || row.job != nil || len(ids) == 0 {
		return
	}

	var change configChange
	var download []banana.File
	for _, o := range m.options(row) {
		if o.file != nil && slices.Contains(ids, o.id) {
			download = append(download, *o.file)
		}
	}
	for _, f := range row.mod.Files {
		kept := false
		for _, v := range f.VPKs {
			if slices.Contains(ids, fmt.Sprintf("vpk:%d", v.ID)) {
				change.enable = append(change.enable, v.ID)
				kept = true
			}
		}
		if !kept {
			change.remove = append(change.remove, f.ID)
		}
	}
	if len(change.enable) == 0 && len(download) == 0 {
		return // none of the ids are the mod's
	}

	if len(download) == 0 {
		if err := m.change(row, change); err != nil {
			slog.Error("failed to change a mod's configuration", "mod", key, "err", err)
			m.notify("Couldn't change the configuration of " + row.mod.Name)
		}
		m.show(row)
		m.publishConfiguration(row, "")
		return
	}
	info := modInfo{
		section: banana.Section(row.mod.Section), id: row.mod.GameBananaID, name: row.mod.Name,
		author: row.mod.Author, thumbnail: row.mod.Thumbnail, updated: row.mod.UpdatedAt, nsfw: row.mod.NSFW,
	}
	m.enqueue(row, &installJob{info: info, files: download, then: &change})
}

// change removes files from an installed mod and picks which of its VPKs are
// used, once any files it needs are installed.
func (m *installedMods) change(row *installedRow, change configChange) error {
	if err := m.db.ConfigureMod(context.Background(), row.mod.ID, change.remove, change.enable); err != nil {
		return err
	}
	var removed []appdb.ModFile
	row.mod.Files = slices.DeleteFunc(row.mod.Files, func(f appdb.ModFile) bool {
		if slices.Contains(change.remove, f.ID) {
			removed = append(removed, f)
			return true
		}
		return false
	})
	for i := range row.mod.Files {
		for j := range row.mod.Files[i].VPKs {
			v := &row.mod.Files[i].VPKs[j]
			v.Enabled = slices.Contains(change.enable, v.ID)
		}
	}
	m.deleteUnusedVPKs(removed)
	slog.Info("changed a mod's configuration", "mod", row.key,
		"removed", len(change.remove), "enabled", len(change.enable))
	return nil
}
