package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/install"
)

// Roles of the installed mods model; their names are in installedRoleNames.
// The ones it shares with the mod listing have the same names, so the same
// components can show either.
const (
	roleInstalledKey = int(qt.UserRole) + 1 + iota
	roleInstalledModID
	roleInstalledName
	roleInstalledAuthor
	roleInstalledThumbnail
	roleInstalledUpdated
	roleInstalledNSFW
	roleInstalledEnabled
	roleInstalledStatus
	roleInstalledProgress
	roleInstalledPace
	roleInstalledError
	roleInstalledVariants
)

var installedRoleNames = map[int][]byte{
	roleInstalledKey:       []byte("key"),
	roleInstalledModID:     []byte("modId"),
	roleInstalledName:      []byte("name"),
	roleInstalledAuthor:    []byte("author"),
	roleInstalledThumbnail: []byte("thumbnail"),
	roleInstalledUpdated:   []byte("updated"),
	roleInstalledNSFW:      []byte("nsfw"),
	roleInstalledEnabled:   []byte("modEnabled"),
	roleInstalledStatus:    []byte("status"),
	roleInstalledProgress:  []byte("progress"),
	roleInstalledPace:      []byte("pace"),
	roleInstalledError:     []byte("error"),
	roleInstalledVariants:  []byte("variants"),
}

// What a mod's row says it's doing (its "status" role).
const (
	statusInstalled   = ""
	statusQueued      = "queued"      // waiting for other installs to finish
	statusDownloading = "downloading" // with its progress, from 0 to 1, and pace
	statusExtracting  = "extracting"
	statusFailed      = "failed" // with the error
)

// How many mods download at once; the rest wait their turn.
const maxRunningInstalls = 2

// installedMods backs the installed mods page: the mods in the database, and
// those being installed, which show their progress until they're done. A
// mod's files are downloaded, checked and added to the store by
// install.Fetch, and only then is it added to the database.
//
// QML lists the mods through installedMods.Filtered, and acts on them by
// writing to these keys:
//
//	query:      lists only the mods with this in their name
//	setEnabled: {key, enabled}: turns a mod on or off
//	cancel:     key: stops installing a mod, or forgets a failed install
//	retry:      key: tries a failed install again
//	uninstall:  key: removes an installed mod
//	configure:  key: fills in "configuration" (see configure) for the mod
//	setConfiguration: {key, ids}: picks which of a mod's options it uses,
//	            by id (see Configure)
//	refresh:    true: looks up which files the installed mods have on
//	            GameBanana, for those it isn't known for yet
//
// A key is a mod's section and id, e.g. "Mod/123", as in its "key" role. Go
// keeps "total" up to date: how many mods there are, whatever the query; and
// "installing": how many are installing or waiting to. As in the clipboard
// bridge, "message" says what just happened, e.g. a download starting, and
// "messageId" goes up by one with each, so QML can tell a repeated message is
// new.
// All methods must be called on the Qt main thread.
type installedMods struct {
	*qml.QQmlPropertyMap
	model *qt.QStandardItemModel
	// What QML lists: the model, narrowed down by the query.
	Filtered *qt.QSortFilterProxyModel

	client *banana.Client
	db     *appdb.DB
	store  install.Store

	rows     map[string]*installedRow // by key
	queue    []*installedRow          // waiting to install, first come first served
	running  int
	messages int

	// The files each installed mod has on GameBanana, by key, once known.
	available map[string][]banana.File
	// The mods whose files are being looked up.
	lookingUp map[string]bool
	// Limits how many are looked up at once.
	lookupSlots chan struct{}
	// The mod whose configuration is in "configuration", if any.
	configuring string
}

// installedRow is one mod on the page.
type installedRow struct {
	key  string
	item *qt.QStandardItem
	mod  *appdb.Mod  // nil until it's installed
	job  *installJob // nil unless it's installing, or failed to
}

// installJob installs some of a mod's files.
type installJob struct {
	info   modInfo
	files  []banana.File
	cancel context.CancelFunc // nil until it starts
	failed bool
	// Changes the mod's configuration once the files are in, if not nil.
	then *configChange
}

// modInfo is what's shown of a mod while it installs, and stored once it has.
type modInfo struct {
	section   banana.Section
	id        int
	name      string
	author    string
	thumbnail string
	updated   time.Time
	nsfw      bool
	// Every file the mod has on GameBanana, if known.
	available []banana.File
}

func (i modInfo) key() string {
	return installedKey(i.section, i.id)
}

func installedKey(section banana.Section, id int) string {
	return fmt.Sprintf("%s/%d", section, id)
}

func newInstalledMods(client *banana.Client, db *appdb.DB, store install.Store) *installedMods {
	m := &installedMods{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		model:           qt.NewQStandardItemModel(),
		Filtered:        qt.NewQSortFilterProxyModel(),
		client:          client,
		db:              db,
		store:           store,
		rows:            make(map[string]*installedRow),
		available:       make(map[string][]banana.File),
		lookingUp:       make(map[string]bool),
		lookupSlots:     make(chan struct{}, 2),
	}
	m.model.SetItemRoleNames(installedRoleNames)
	m.Filtered.SetSourceModel(m.model.QAbstractItemModel)
	m.Filtered.SetFilterRole(roleInstalledName)
	m.Filtered.SetFilterCaseSensitivity(qt.CaseInsensitive)

	m.Insert("query", qt.NewQVariant14(""))
	for _, key := range []string{"cancel", "retry", "uninstall"} {
		m.Insert(key, qt.NewQVariant14(""))
	}
	m.Insert("setEnabled", qt.NewQVariant20(map[string]qt.QVariant{}))
	m.Insert("setConfiguration", qt.NewQVariant20(map[string]qt.QVariant{}))
	m.Insert("refresh", qt.NewQVariant8(false))
	m.Insert("configure", qt.NewQVariant14(""))
	m.Insert("configuration", qt.NewQVariant20(map[string]qt.QVariant{}))
	m.Insert("message", qt.NewQVariant14(""))
	m.Insert("messageId", qt.NewQVariant4(0))

	mods, err := db.Mods(context.Background())
	if err != nil {
		slog.Error("failed to load the installed mods", "err", err)
	}
	for i := range mods {
		row := m.addRow(installedKey(banana.Section(mods[i].Section), mods[i].GameBananaID))
		row.mod = &mods[i]
		m.show(row)
	}
	m.updateCounts()

	// Only fires for changes made from QML, not for our own Inserts.
	m.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "query":
			m.Filtered.SetFilterFixedString(value.ToString())
			return
		case "setEnabled":
			request := value.ToMap()
			keyValue, enabledValue := request["key"], request["enabled"]
			m.SetEnabled(keyValue.ToString(), enabledValue.ToBool())
			m.Insert(key, qt.NewQVariant20(map[string]qt.QVariant{}))
			return
		case "setConfiguration":
			request := value.ToMap()
			keyValue, idsValue := request["key"], request["ids"]
			var ids []string
			for _, id := range idsValue.ToList() {
				ids = append(ids, id.ToString())
			}
			m.Configure(keyValue.ToString(), ids)
			m.Insert(key, qt.NewQVariant20(map[string]qt.QVariant{}))
			return
		case "refresh":
			if value.ToBool() {
				m.Insert(key, qt.NewQVariant8(false))
				m.lookUpAll()
			}
			return
		}

		rowKey := value.ToString()
		if rowKey == "" {
			return
		}
		// Clear the key, so asking for the same thing again is still a change.
		m.Insert(key, qt.NewQVariant14(""))
		switch key {
		case "cancel":
			m.Cancel(rowKey)
		case "retry":
			m.Retry(rowKey)
		case "uninstall":
			m.Uninstall(rowKey)
		case "configure":
			m.configure(rowKey)
		}
	})
	return m
}

// Install downloads and installs a mod's files, after any installs already
// waiting. Files that are already installed are skipped, and so is a mod
// that's already installing.
func (m *installedMods) Install(info modInfo, files []banana.File) {
	if info.available != nil {
		m.available[info.key()] = info.available
	}
	row := m.rows[info.key()]
	if row != nil && row.job != nil && !row.job.failed {
		slog.Info("already installing this mod", "mod", row.key)
		m.notify(info.name + " is already downloading")
		return
	}
	if row != nil && row.mod != nil {
		files = notInstalled(row.mod, files)
	}
	if len(files) == 0 {
		slog.Info("nothing to install: the files are installed already", "mod", info.key())
		m.notify(info.name + " is already installed")
		return
	}

	if row == nil {
		row = m.addRow(info.key())
	}
	m.enqueue(row, &installJob{info: info, files: files})
}

// notInstalled returns the files that aren't among mod's files already.
func notInstalled(mod *appdb.Mod, files []banana.File) []banana.File {
	have := make(map[int]bool, len(mod.Files))
	for _, f := range mod.Files {
		have[f.GameBananaID] = true
	}
	var missing []banana.File
	for _, f := range files {
		if !have[f.ID] {
			missing = append(missing, f)
		}
	}
	return missing
}

func (m *installedMods) enqueue(row *installedRow, job *installJob) {
	row.job = job
	m.queue = append(m.queue, row)
	m.show(row)
	m.updateCounts()
	m.startNext()
	if slices.Contains(m.queue, row) {
		m.notify("Waiting to download " + job.info.name)
	}
}

// startNext starts the installs that are waiting, as long as there's room.
func (m *installedMods) startNext() {
	for m.running < maxRunningInstalls && len(m.queue) > 0 {
		row := m.queue[0]
		m.queue = m.queue[1:]
		m.start(row)
	}
}

func (m *installedMods) start(row *installedRow) {
	job := row.job
	ctx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel
	m.running++
	m.showStatus(row, install.Status{Stage: install.Downloading})
	m.notify("Downloading " + job.info.name)

	slog.Info("installing a mod", "mod", row.key, "files", len(job.files))
	go func() {
		files, err := install.Fetch(ctx, m.client, m.store, job.files, func(s install.Status) {
			mainthread.Start(func() {
				if row.job == job {
					m.showStatus(row, s)
				}
			})
		})
		mainthread.Start(func() {
			cancel()
			m.running--
			m.finish(row, job, files, err)
			m.startNext()
		})
	}()
}

// finish records an install that has stopped: it either installed its files
// or failed to. A job that's been cancelled has been dealt with already.
func (m *installedMods) finish(row *installedRow, job *installJob, files []appdb.ModFile, err error) {
	if row.job != job {
		return // cancelled
	}
	defer m.updateCounts()
	if err == nil {
		err = m.record(row, job.info, files)
	}
	if err != nil {
		slog.Error("failed to install a mod", "mod", row.key, "err", err)
		job.failed = true
		row.item.SetData(qt.NewQVariant14(statusFailed), roleInstalledStatus)
		row.item.SetData(qt.NewQVariant14(err.Error()), roleInstalledError)
		return
	}
	slog.Info("installed a mod", "mod", row.key)
	row.job = nil
	m.show(row)
	m.lookUp(row)
}

// record adds a mod's newly installed files to the database, along with the
// mod if it's new.
func (m *installedMods) record(row *installedRow, info modInfo, files []appdb.ModFile) error {
	ctx := context.Background()
	if row.mod != nil {
		added, err := m.db.AddModFiles(ctx, row.mod.ID, files)
		if err != nil {
			return err
		}
		row.mod.Files = append(row.mod.Files, added...)
		if then := row.job.then; then != nil {
			for _, f := range added {
				for _, v := range f.VPKs {
					then.enable = append(then.enable, v.ID)
				}
			}
			return m.change(row, *then)
		}
		return nil
	}

	mod, err := m.db.AddMod(ctx, appdb.Mod{
		Section:      string(info.section),
		GameBananaID: info.id,
		Name:         info.name,
		Author:       info.author,
		Enabled:      true,
		LoadOrder:    m.nextLoadOrder(),
		InstalledAt:  time.Now(),
		Thumbnail:    info.thumbnail,
		UpdatedAt:    info.updated,
		NSFW:         info.nsfw,
		Files:        files,
	})
	if err != nil {
		return err
	}
	row.mod = &mod
	return nil
}

// nextLoadOrder returns the load order for a new mod: after every other.
func (m *installedMods) nextLoadOrder() int {
	next := 0
	for _, row := range m.rows {
		if row.mod != nil {
			next = max(next, row.mod.LoadOrder+1)
		}
	}
	return next
}

// Cancel stops installing a mod, or forgets an install that failed. A mod
// that wasn't installed before goes away; one that was stays as it was.
func (m *installedMods) Cancel(key string) {
	row := m.rows[key]
	if row == nil || row.job == nil {
		return
	}
	if row.job.cancel != nil {
		row.job.cancel()
	}
	row.job = nil
	for i, queued := range m.queue {
		if queued == row {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}
	slog.Info("cancelled installing a mod", "mod", key)
	if row.mod == nil {
		m.removeRow(row)
		return
	}
	m.show(row)
	m.updateCounts()
}

// Retry tries a failed install again.
func (m *installedMods) Retry(key string) {
	row := m.rows[key]
	if row == nil || row.job == nil || !row.job.failed {
		return
	}
	m.enqueue(row, &installJob{info: row.job.info, files: row.job.files, then: row.job.then})
}

// SetEnabled turns an installed mod on or off.
func (m *installedMods) SetEnabled(key string, enabled bool) {
	row := m.rows[key]
	if row == nil || row.mod == nil || row.mod.Enabled == enabled {
		return
	}
	if err := m.db.SetModEnabled(context.Background(), row.mod.ID, enabled); err != nil {
		slog.Error("failed to turn a mod on or off", "mod", key, "err", err)
		return
	}
	row.mod.Enabled = enabled
	row.item.SetData(qt.NewQVariant8(enabled), roleInstalledEnabled)
}

// Uninstall removes an installed mod, and deletes its VPKs from the store
// unless another mod has them too.
func (m *installedMods) Uninstall(key string) {
	row := m.rows[key]
	if row == nil || row.mod == nil || row.job != nil {
		return
	}
	ctx := context.Background()
	if err := m.db.RemoveMod(ctx, row.mod.ID); err != nil {
		slog.Error("failed to uninstall a mod", "mod", key, "err", err)
		return
	}
	slog.Info("uninstalled a mod", "mod", key)
	m.removeRow(row)

	m.deleteUnusedVPKs(row.mod.Files)
}

// deleteUnusedVPKs deletes the VPKs of files that have been removed from the
// database from the store, unless another mod has them too.
func (m *installedMods) deleteUnusedVPKs(files []appdb.ModFile) {
	// A running install may have just added the same VPK to the store, and
	// not recorded it yet. Unused VPKs are cleared out on startup anyway.
	if m.running > 0 {
		return
	}
	ctx := context.Background()
	for _, f := range files {
		for _, v := range f.VPKs {
			if inUse, err := m.db.VPKInUse(ctx, v.SHA256); err != nil || inUse {
				continue
			}
			if err := m.store.Remove(v.SHA256); err != nil {
				slog.Error("failed to delete a VPK that's no longer used", "sha256", v.SHA256, "err", err)
			}
		}
	}
}

// notify says what just happened, e.g. that a download has started.
func (m *installedMods) notify(message string) {
	m.messages++
	m.Insert("message", qt.NewQVariant14(message))
	m.Insert("messageId", qt.NewQVariant4(m.messages))
}

func (m *installedMods) addRow(key string) *installedRow {
	item := qt.NewQStandardItem()
	item.SetEditable(false)
	item.SetData(qt.NewQVariant14(key), roleInstalledKey)
	m.model.AppendRow([]*qt.QStandardItem{item})
	row := &installedRow{key: key, item: item}
	m.rows[key] = row
	return row
}

func (m *installedMods) removeRow(row *installedRow) {
	delete(m.rows, row.key)
	m.model.RemoveRow(row.item.Row())
	m.updateCounts()
}

func (m *installedMods) updateCounts() {
	installing := 0
	for _, row := range m.rows {
		if row.job != nil && !row.job.failed {
			installing++
		}
	}
	m.Insert("total", qt.NewQVariant4(len(m.rows)))
	m.Insert("installing", qt.NewQVariant4(installing))
}

// show fills in a row: from the database once the mod is installed, and from
// what GameBanana said about it until then.
func (m *installedMods) show(row *installedRow) {
	item := row.item
	if mod := row.mod; mod != nil {
		item.SetData(qt.NewQVariant4(mod.GameBananaID), roleInstalledModID)
		item.SetData(qt.NewQVariant14(mod.Name), roleInstalledName)
		item.SetData(qt.NewQVariant14(mod.Author), roleInstalledAuthor)
		item.SetData(qt.NewQVariant14(mod.Thumbnail), roleInstalledThumbnail)
		item.SetData(qt.NewQVariant6(mod.UpdatedAt.Unix()), roleInstalledUpdated)
		item.SetData(qt.NewQVariant8(mod.NSFW), roleInstalledNSFW)
		item.SetData(qt.NewQVariant8(mod.Enabled), roleInstalledEnabled)
		item.SetData(qt.NewQVariant4(len(m.options(row))), roleInstalledVariants)
	} else if row.job != nil {
		info := row.job.info
		item.SetData(qt.NewQVariant4(info.id), roleInstalledModID)
		item.SetData(qt.NewQVariant14(info.name), roleInstalledName)
		item.SetData(qt.NewQVariant14(info.author), roleInstalledAuthor)
		item.SetData(qt.NewQVariant14(info.thumbnail), roleInstalledThumbnail)
		item.SetData(qt.NewQVariant6(info.updated.Unix()), roleInstalledUpdated)
		item.SetData(qt.NewQVariant8(info.nsfw), roleInstalledNSFW)
		// New mods are turned on once they're installed.
		item.SetData(qt.NewQVariant8(true), roleInstalledEnabled)
		item.SetData(qt.NewQVariant4(0), roleInstalledVariants)
	}

	status := statusInstalled
	if row.job != nil {
		status = statusQueued
	}
	item.SetData(qt.NewQVariant14(status), roleInstalledStatus)
	item.SetData(qt.NewQVariant9(0), roleInstalledProgress)
	item.SetData(qt.NewQVariant14(install.PaceNormal.String()), roleInstalledPace)
	item.SetData(qt.NewQVariant14(""), roleInstalledError)
}

// showStatus shows how a running install is going.
func (m *installedMods) showStatus(row *installedRow, s install.Status) {
	status := statusDownloading
	if s.Stage == install.Extracting {
		status = statusExtracting
	}
	row.item.SetData(qt.NewQVariant14(status), roleInstalledStatus)
	row.item.SetData(qt.NewQVariant9(s.Progress()), roleInstalledProgress)
	row.item.SetData(qt.NewQVariant14(s.Pace.String()), roleInstalledPace)
}

// removeUnusedVPKs deletes the VPKs in the store that no installed mod has,
// e.g. from installs that were cancelled, or that the app quit during. It
// must only run while nothing is installing, and only on the database the
// store belongs to: on one that's empty because the real one couldn't be
// opened, it would delete every VPK.
func removeUnusedVPKs(ctx context.Context, db *appdb.DB, store install.Store) error {
	shas, err := store.VPKs()
	if err != nil {
		return err
	}
	var errs []error
	for _, sha := range shas {
		inUse, err := db.VPKInUse(ctx, sha)
		if err != nil {
			return err
		}
		if !inUse {
			slog.Info("deleting a VPK that no installed mod has", "sha256", sha)
			errs = append(errs, store.Remove(sha))
		}
	}
	return errors.Join(errs...)
}
