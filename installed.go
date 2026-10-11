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

// The roles shared with the mod listing have the same names, so the same
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
	roleInstalledErrorDetail
	roleInstalledVariants
)

var installedRoleNames = map[int][]byte{
	roleInstalledKey:         []byte("key"),
	roleInstalledModID:       []byte("modId"),
	roleInstalledName:        []byte("name"),
	roleInstalledAuthor:      []byte("author"),
	roleInstalledThumbnail:   []byte("thumbnail"),
	roleInstalledUpdated:     []byte("updated"),
	roleInstalledNSFW:        []byte("nsfw"),
	roleInstalledEnabled:     []byte("modEnabled"),
	roleInstalledStatus:      []byte("status"),
	roleInstalledProgress:    []byte("progress"),
	roleInstalledPace:        []byte("pace"),
	roleInstalledError:       []byte("error"),
	roleInstalledErrorDetail: []byte("errorDetail"),
	roleInstalledVariants:    []byte("variants"),
}

// The "status" role.
const (
	statusInstalled   = ""
	statusQueued      = "queued"
	statusDownloading = "downloading" // with its progress, from 0 to 1, and pace
	statusExtracting  = "extracting"
	statusFailed      = "failed" // with the error
)

const maxRunningInstalls = 2

// installedMods backs the installed mods page. A mod being installed shows its
// progress until it's done, and is only added to the database once
// install.Fetch has downloaded and checked its files and added them to the
// store.
//
// QML lists the mods through installedMods.Filtered, and acts on them by
// writing to these keys:
//
//	query:      lists only the mods with this in their name
//	setEnabled: {key, enabled}: turns a mod on or off
//	cancel:     key: stops installing a mod, or forgets a failed install
//	retry:      key: tries a failed install again
//	uninstall:  key: removes an installed mod, which can be undone until
//	            its message goes
//	undo:       messageId: puts back the mod uninstalled with that message
//	messageGone: messageId: that message is no longer shown, so its undo
//	            is too late
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
// new. "messageUndoable" says whether the message offers Undo.
// All methods must be called on the Qt main thread.
type installedMods struct {
	*qml.QQmlPropertyMap
	model    *qt.QStandardItemModel
	Filtered *qt.QSortFilterProxyModel

	client *banana.Client
	db     *appdb.DB
	store  install.Store

	rows     map[string]*installedRow // by key
	queue    []*installedRow          // first come first served
	running  int
	messages int

	// The files each installed mod has on GameBanana, by key, once known.
	available map[string][]banana.File
	lookingUp map[string]bool
	// Limits how many mods are looked up at once.
	lookupSlots chan struct{}
	// The mod whose configuration is in "configuration", if any.
	configuring string
	// An uninstall that can still be undone. The mod stays in the database
	// until it can't.
	uninstalling *pendingUninstall

	// Called when an install fails, and when the last install still going
	// finishes. Either may be nil.
	onFailed func(name string)
	onIdle   func()
}

type pendingUninstall struct {
	row *installedRow
	// Where the row was in the model, to put it back there.
	index     int
	messageID int
}

type installedRow struct {
	key  string
	item *qt.QStandardItem
	mod  *appdb.Mod  // nil until it's installed
	job  *installJob // nil unless it's installing, or failed to
}

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
	m.Insert("messageUndoable", qt.NewQVariant8(false))
	m.Insert("messageId", qt.NewQVariant4(0))
	m.Insert("undo", qt.NewQVariant4(0))
	m.Insert("messageGone", qt.NewQVariant4(0))

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
		case "undo":
			m.Undo(value.ToInt())

			return
		case "messageGone":
			m.messageGone(value.ToInt())

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

// Install queues behind any installs already waiting. Files that are already
// installed are skipped, and so is a mod that's already installing.
func (m *installedMods) Install(info modInfo, files []banana.File) {
	// Reinstalling a mod while its uninstall can still be undone starts from
	// it being gone.
	if m.uninstalling != nil && m.uninstalling.row.key == info.key() {
		m.FinishUninstall()
	}

	if info.available != nil {
		m.available[info.key()] = info.available
	}

	row := m.rows[info.key()]
	if row != nil && row.job != nil && !row.job.failed {
		slog.Info("already installing this mod", "mod", row.key)
		m.notify(fmt.Sprintf(tr("%s is already downloading"), info.name))

		return
	}

	if row != nil && row.mod != nil {
		files = notInstalled(row.mod, files)
	}

	if len(files) == 0 {
		slog.Info("nothing to install: the files are installed already", "mod", info.key())
		m.notify(fmt.Sprintf(tr("%s is already installed"), info.name))

		return
	}

	if row == nil {
		row = m.addRow(info.key())
	}

	m.enqueue(row, &installJob{info: info, files: files})
}

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

	// It's still in the queue if there wasn't room to start it.
	if slices.Contains(m.queue, row) {
		m.notify(fmt.Sprintf(tr("Waiting to download %s"), job.info.name))
	}
}

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
	m.notify(fmt.Sprintf(tr("Downloading %s"), job.info.name))

	slog.Info("installing a mod", "mod", row.key, "files", len(job.files))

	go func() {
		files, err := install.Fetch(ctx, m.client, m.store, job.files, func(s install.Status) {
			mainthread.Start(func() {
				// Unless it's been cancelled since.
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

			if m.Idle() && m.onIdle != nil {
				m.onIdle()
			}
		})
	}()
}

// Idle is true when nothing is installing or waiting to.
func (m *installedMods) Idle() bool {
	return m.running == 0 && len(m.queue) == 0
}

// A job that's been cancelled has been dealt with already.
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
		message, detail := errorTexts(err)
		row.item.SetData(qt.NewQVariant14(message), roleInstalledError)
		row.item.SetData(qt.NewQVariant14(detail), roleInstalledErrorDetail)

		if m.onFailed != nil {
			m.onFailed(job.info.name)
		}

		return
	}

	slog.Info("installed a mod", "mod", row.key)

	row.job = nil
	m.show(row)
	m.lookUp(row)
}

func (m *installedMods) record(row *installedRow, info modInfo, files []appdb.ModFile) error {
	ctx := context.Background()

	if row.mod != nil {
		added, err := m.db.AddModFiles(ctx, row.mod.ID, files)
		if err != nil {
			return err
		}

		row.mod.Files = append(row.mod.Files, added...)

		// A configuration change that was waiting for these files uses their
		// VPKs too.
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

func (m *installedMods) nextLoadOrder() int {
	next := 0

	for _, row := range m.rows {
		if row.mod != nil {
			next = max(next, row.mod.LoadOrder+1)
		}
	}

	return next
}

// Cancel also forgets an install that failed. A mod that wasn't installed
// before goes away; one that was stays as it was.
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

func (m *installedMods) Retry(key string) {
	row := m.rows[key]
	if row == nil || row.job == nil || !row.job.failed {
		return
	}

	m.enqueue(row, &installJob{info: row.job.info, files: row.job.files, then: row.job.then})
}

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

// Uninstall takes the mod off the list straight away, but only removes it
// once the message offering Undo has gone (see messageGone), or on
// FinishUninstall.
func (m *installedMods) Uninstall(key string) {
	row := m.rows[key]
	if row == nil || row.mod == nil || row.job != nil {
		return
	}

	// Only the latest uninstall can be undone.
	m.FinishUninstall()

	// Take the row out without deleting its item, so Undo can put it back.
	index := row.item.Row()
	m.model.TakeRow(index)
	delete(m.rows, key)
	m.updateCounts()

	m.notifyUndoable(fmt.Sprintf(tr("%s was uninstalled"), row.mod.Name))
	m.uninstalling = &pendingUninstall{row: row, index: index, messageID: m.messages}
}

func (m *installedMods) Undo(messageID int) {
	p := m.uninstalling
	if p == nil || p.messageID != messageID {
		return
	}

	m.uninstalling = nil
	m.putBack(p)

	slog.Info("undid uninstalling a mod", "mod", p.row.key)
}

func (m *installedMods) messageGone(messageID int) {
	if m.uninstalling != nil && m.uninstalling.messageID == messageID {
		m.FinishUninstall()
	}
}

// FinishUninstall removes the mod whose uninstall could still be undone, if
// any. Call it before quitting.
func (m *installedMods) FinishUninstall() {
	p := m.uninstalling
	if p == nil {
		return
	}

	m.uninstalling = nil

	if err := m.db.RemoveMod(context.Background(), p.row.mod.ID); err != nil {
		slog.Error("failed to uninstall a mod", "mod", p.row.key, "err", err)

		// It's still installed, so it goes back on the list.
		m.putBack(p)
		m.notify(fmt.Sprintf(tr("Couldn’t uninstall %s"), p.row.mod.Name))

		return
	}

	slog.Info("uninstalled a mod", "mod", p.row.key)
	p.row.item.Delete()

	m.deleteUnusedVPKs(p.row.mod.Files)
}

func (m *installedMods) putBack(p *pendingUninstall) {
	m.model.InsertRow(min(p.index, m.model.RowCount(qt.NewQModelIndex())), []*qt.QStandardItem{p.row.item})
	m.rows[p.row.key] = p.row
	m.updateCounts()
}

// The files must have been removed from the database already. VPKs another
// mod has too are kept.
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

func (m *installedMods) notify(message string) {
	m.sendMessage(message, false)
}

func (m *installedMods) notifyUndoable(message string) {
	m.sendMessage(message, true)
}

func (m *installedMods) sendMessage(message string, undoable bool) {
	m.messages++
	m.Insert("message", qt.NewQVariant14(message))
	m.Insert("messageUndoable", qt.NewQVariant8(undoable))
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

// show fills in a row from the database once the mod is installed, and from
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
	item.SetData(qt.NewQVariant14(""), roleInstalledErrorDetail)
}

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
