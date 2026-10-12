package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
)

// sectionLocal is the section of a mod that isn't from GameBanana, e.g. a
// VPK taken over from another tool.
const sectionLocal banana.Section = "Local"

// otherMods backs the installed page's "From Other Tools" section: the mods
// in the game's addons folders that another tool, or the user, put there
// (see deploy.ScanOtherMods). They're listed in a model with the roles of
// installedMods, so the same views show them, with "author" naming where
// they're from. Lockdeader only shows them, until asked to take them over.
// Then a mod GameBanana knows is downloaded from there and installed like
// any other, so it's no different from one installed here; a file it
// doesn't know becomes a local mod, as it is. Either way the files leave
// the addons folder once the mod is in, and Deadlock Mod Manager's ledger
// forgets it.
//
// QML asks for things by writing keys:
//
//	scan:     true: looks through the addons folders again
//	takeOver: [{key, ids}]: takes every mod listed over, downloading the
//	          files with these ids (see "choices") for the mods with
//	          several to choose from; a mod left out gets its default
//
// Go fills in "count", how many there are; "dmmCount", how many of them
// Deadlock Mod Manager's; "scanning", "lookingUp" (what they are on
// GameBanana) and "takingOver" while it is; "folderUrl", the addons folder,
// for opening it; and "choices", the mods with several files on GameBanana
// to pick from: {key, name, thumbnail, files, picked} objects, files as
// details.files and picked the id of the file that's probably the one. What
// a take-over came to is said through installed's message.
//
// All methods must be called on the Qt main thread.
type otherMods struct {
	*qml.QQmlPropertyMap
	Model *qt.QStandardItemModel

	client    *banana.Client
	store     install.Store
	installed *installedMods
	gameDir   func() string

	// deploy.ScanOtherMods, other than in tests.
	scan func(gameDir string) ([]deploy.OtherMod, error)

	rows []*otherRow
	// What GameBanana says about a mod, by DMM key, kept across scans.
	profiles  map[string]*lookedUp
	lookingUp map[string]bool
	// Limits how many mods are looked up at once.
	lookupSlots chan struct{}

	scanning, scanAgain, takingOver bool
}

type otherRow struct {
	mod  deploy.OtherMod
	item *qt.QStandardItem
}

type lookedUp struct {
	section banana.Section
	profile *banana.ModProfile // nil when it couldn't be found
}

func newOtherMods(client *banana.Client, store install.Store, installed *installedMods, gameDir func() string) *otherMods {
	o := &otherMods{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		Model:           qt.NewQStandardItemModel(),
		client:          client,
		store:           store,
		installed:       installed,
		gameDir:         gameDir,
		scan:            deploy.ScanOtherMods,
		profiles:        make(map[string]*lookedUp),
		lookingUp:       make(map[string]bool),
		lookupSlots:     make(chan struct{}, 2),
	}

	o.Model.SetItemRoleNames(installedRoleNames)

	o.Insert("scan", qt.NewQVariant8(false))
	o.Insert("takeOver", qt.NewQVariant8(false))
	o.Insert("count", qt.NewQVariant4(0))
	o.Insert("dmmCount", qt.NewQVariant4(0))
	o.Insert("scanning", qt.NewQVariant8(false))
	o.Insert("lookingUp", qt.NewQVariant8(false))
	o.Insert("takingOver", qt.NewQVariant8(false))
	o.Insert("folderUrl", qt.NewQVariant14(""))
	o.Insert("choices", qt.NewQVariant43(nil))

	// Only fires for changes made from QML, not for our own Inserts.
	o.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "scan":
			if value.ToBool() {
				o.Insert(key, qt.NewQVariant8(false))
				o.Scan()
			}
		case "takeOver":
			picks := make(map[string][]int)

			for _, pick := range value.ToList() {
				m := pick.ToMap()
				keyValue, idsValue := m["key"], m["ids"]

				var ids []int

				for _, id := range idsValue.ToList() {
					ids = append(ids, id.ToInt())
				}

				picks[keyValue.ToString()] = ids
			}

			o.Insert(key, qt.NewQVariant8(false))
			o.TakeOver(picks)
		}
	})

	return o
}

// Scan lists the mods in the addons folders again. With no game folder,
// there are none.
func (o *otherMods) Scan() {
	dir := o.gameDir()
	if dir == "" {
		o.show(nil)

		return
	}

	if o.scanning {
		o.scanAgain = true

		return
	}

	o.scanning = true
	o.Insert("scanning", qt.NewQVariant8(true))

	go func() {
		mods, err := o.scan(dir)

		mainthread.Start(func() {
			o.scanning = false
			o.Insert("scanning", qt.NewQVariant8(false))

			if err != nil {
				slog.Error("failed to look through the game's addons folders", "err", err)
			} else {
				o.show(mods)
			}

			if o.scanAgain {
				o.scanAgain = false
				o.Scan()
			}
		})
	}()
}

func (o *otherMods) show(mods []deploy.OtherMod) {
	o.Model.Clear()
	o.rows = nil
	dmm := 0

	for _, m := range mods {
		item := qt.NewQStandardItem()
		item.SetEditable(false)
		o.Model.AppendRow([]*qt.QStandardItem{item})

		row := &otherRow{mod: m, item: item}
		o.rows = append(o.rows, row)
		o.showRow(row)

		if m.Source == deploy.SourceDMM {
			dmm++
		}

		if m.GameBananaID != 0 {
			o.lookUp(m)
		}
	}

	o.Insert("count", qt.NewQVariant4(len(mods)))
	o.Insert("dmmCount", qt.NewQVariant4(dmm))
	o.publishChoices()

	url := ""

	if dir := o.gameDir(); dir != "" {
		url = qt.QUrl_FromLocalFile(deploy.AddonsFolder(dir)).ToString()
	}

	o.Insert("folderUrl", qt.NewQVariant14(url))
}

func (o *otherMods) showRow(row *otherRow) {
	m, item := row.mod, row.item
	key, name, author, thumbnail, updated := m.Key, "", "", "", m.Modified

	var names []string

	for _, f := range m.Files {
		names = append(names, f.Name)
	}

	name = strings.Join(names, ", ")

	switch m.Source {
	case deploy.SourceDMM:
		author = tr("Deadlock Mod Manager")
	default:
		author = tr("Addons folder")
	}

	if l := o.profiles[m.DMMKey]; l != nil && l.profile != nil {
		// Sounds have no picture on GameBanana; the views give them the
		// app's by the key.
		key = installedKey(l.section, l.profile.ID)
		name = l.profile.Name
		thumbnail = previewURL(l.profile.PreviewMedia)
		updated = l.profile.LastUpdated()
	}

	item.SetData(qt.NewQVariant14(key), roleInstalledKey)
	item.SetData(qt.NewQVariant4(m.GameBananaID), roleInstalledModID)
	item.SetData(qt.NewQVariant14(name), roleInstalledName)
	item.SetData(qt.NewQVariant14(author), roleInstalledAuthor)
	item.SetData(qt.NewQVariant14(thumbnail), roleInstalledThumbnail)
	item.SetData(qt.NewQVariant6(updated.Unix()), roleInstalledUpdated)
	item.SetData(qt.NewQVariant8(false), roleInstalledNSFW)
	item.SetData(qt.NewQVariant8(m.Enabled), roleInstalledEnabled)
	item.SetData(qt.NewQVariant14(statusInstalled), roleInstalledStatus)
	item.SetData(qt.NewQVariant9(0), roleInstalledProgress)
	item.SetData(qt.NewQVariant14(install.PaceNormal.String()), roleInstalledPace)
	item.SetData(qt.NewQVariant14(""), roleInstalledError)
	item.SetData(qt.NewQVariant14(""), roleInstalledErrorDetail)
	item.SetData(qt.NewQVariant4(0), roleInstalledVariants)
}

// lookUp asks GameBanana what a mod is, by the id in Deadlock Mod Manager's
// ledger. Older ledgers don't say whether an id is a mod's or a sound's, so
// a mod by that id that isn't Deadlock's means it's a sound.
func (o *otherMods) lookUp(m deploy.OtherMod) {
	key := m.DMMKey
	if o.profiles[key] != nil || o.lookingUp[key] {
		return
	}

	o.lookingUp[key] = true
	o.Insert("lookingUp", qt.NewQVariant8(true))

	go func() {
		o.lookupSlots <- struct{}{}
		result := o.find(m.GameBananaID, banana.Section(m.Section))
		<-o.lookupSlots

		mainthread.Start(func() {
			delete(o.lookingUp, key)
			o.Insert("lookingUp", qt.NewQVariant8(len(o.lookingUp) > 0))

			if result == nil {
				return // try again at the next scan
			}

			o.profiles[key] = result

			for _, row := range o.rows {
				if row.mod.DMMKey == key {
					o.showRow(row)
				}
			}

			o.publishChoices()
		})
	}()
}

// publishChoices fills in "choices": the mods that have several files on
// GameBanana, for the take-over dialog to ask which to download.
func (o *otherMods) publishChoices() {
	var choices []qt.QVariant

	for _, row := range o.rows {
		l := o.profiles[row.mod.DMMKey]
		if l == nil || l.profile == nil || len(l.profile.Files) < 2 {
			continue
		}

		choices = append(choices, *qt.NewQVariant20(map[string]qt.QVariant{
			"key":       *qt.NewQVariant14(row.mod.Key),
			"name":      *qt.NewQVariant14(l.profile.Name),
			"thumbnail": *qt.NewQVariant14(previewURL(l.profile.PreviewMedia)),
			"files":     *fileList(l.profile.Files),
			"picked":    *qt.NewQVariant4(defaultPick(row.mod, l.profile.Files).ID),
		}))
	}

	o.Insert("choices", qt.NewQVariant43(choices))
}

// defaultPick guesses which of a mod's files on GameBanana the other tool
// installed: the one named like one of the VPKs (rosia.zip for rosia.vpk),
// or failing that the first, which is the one most mods want.
func defaultPick(m deploy.OtherMod, files []banana.File) banana.File {
	for _, f := range m.Files {
		stem := strings.ToLower(strings.TrimSuffix(f.Name, ".vpk"))

		for _, file := range files {
			name := strings.ToLower(strings.TrimSuffix(file.Name, filepathExt(file.Name)))

			if name == stem || strings.HasPrefix(name, stem+"_") {
				return file
			}
		}
	}

	return files[0]
}

func filepathExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}

	return ""
}

// find returns nil if GameBanana couldn't be asked, and a result with no
// profile if the mod isn't there.
func (o *otherMods) find(id int, section banana.Section) *lookedUp {
	ctx := context.Background()
	sections := []banana.Section{section}

	if section == "" {
		sections = []banana.Section{banana.SectionMods, banana.SectionSounds}
	}

	for _, s := range sections {
		p, err := o.client.GetModProfile(ctx, s, id)

		var apiErr *banana.APIError

		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			continue
		}

		if err != nil {
			slog.Error("failed to look up a mod from another tool", "section", s, "id", id, "err", err)

			return nil
		}

		if p.Game.ID == banana.DeadlockGameID {
			return &lookedUp{section: s, profile: p}
		}
	}

	return &lookedUp{}
}

// TakeOver moves every mod listed into Lockdeader. A mod GameBanana knows
// is downloaded from there, like any install, with the files picked for it
// (see "choices"); its files in the addons folder go once it's in, so a
// download that fails loses nothing. A file GameBanana doesn't know goes
// into the store as it is, as a local mod. A mod that's already installed
// from GameBanana stays where it is. Each ends up on or off as it was.
func (o *otherMods) TakeOver(picks map[string][]int) {
	dir := o.gameDir()
	if dir == "" || o.takingOver || len(o.rows) == 0 {
		return
	}

	o.takingOver = true
	o.Insert("takingOver", qt.NewQVariant8(true))

	pending, took, kept := 0, 0, 0

	// finished is called once per mod, on the main thread, as each is done.
	finished := func(ok bool) {
		if ok {
			took++
		} else {
			kept++
		}

		pending--

		if pending > 0 {
			return
		}

		o.takingOver = false
		o.Insert("takingOver", qt.NewQVariant8(false))
		o.Scan()

		switch {
		case took == 0:
			o.installed.notify(tr("Couldn’t take the mods over. They’re still where they were."))
		case kept == 0 && took == 1:
			o.installed.notify(tr("The mod was taken over"))
		case kept == 0:
			o.installed.notify(trN("%n mod was taken over", "%n mods were taken over", took))
		default:
			//: %d is how many mods weren't taken over.
			o.installed.notify(fmt.Sprintf(trN("%n mod was taken over; %d couldn’t be, or are installed already",
				"%n mods were taken over; %d couldn’t be, or are installed already", took), kept))
		}
	}

	var local []deploy.OtherMod

	for _, row := range o.rows {
		m := row.mod
		l := o.profiles[m.DMMKey]

		if l == nil || l.profile == nil {
			local = append(local, m)

			continue
		}

		files := pickFiles(l.profile.Files, picks[m.Key])
		if len(files) == 0 {
			files = []banana.File{defaultPick(m, l.profile.Files)}
		}

		info := modInfo{
			section:   l.section,
			id:        l.profile.ID,
			name:      l.profile.Name,
			author:    l.profile.Submitter.Name,
			thumbnail: previewURL(l.profile.PreviewMedia),
			updated:   l.profile.LastUpdated(),
			available: l.profile.Files,
		}

		pending++

		queued := o.installed.InstallThen(info, files, func(err error) {
			if err != nil {
				finished(false)

				return
			}

			if !m.Enabled {
				o.installed.SetEnabled(info.key(), false)
			}

			o.removeTaken(dir, m)
			finished(true)
		})

		if !queued {
			pending--
			kept++
		}
	}

	pending += len(local)

	if pending == 0 {
		// Nothing was queued: every mod is installed already.
		pending = 1
		finished(false)

		return
	}

	if len(local) == 0 {
		return
	}

	type taken struct {
		mod   deploy.OtherMod
		files []appdb.ModFile
		err   error
	}

	go func() {
		results := make([]taken, len(local))

		for i, m := range local {
			results[i] = taken{mod: m, files: []appdb.ModFile{{
				Name: strings.Join(fileNames(m), ", "), UploadedAt: m.Modified, DownloadedAt: time.Now(),
			}}}

			for _, f := range m.Files {
				vpks, err := install.Extract(f.Path, f.Name, o.store)
				if err != nil {
					results[i].err = err

					break
				}

				file := &results[i].files[0]
				file.Size += f.Size

				for _, v := range vpks {
					file.VPKs = append(file.VPKs, appdb.VPK{Path: f.Name, SHA256: v.SHA256, Size: v.Size, Enabled: f.Enabled || !m.Enabled})
				}
			}
		}

		mainthread.Start(func() {
			for _, r := range results {
				if r.err != nil {
					slog.Error("failed to take a mod over", "mod", r.mod.Key, "err", r.err)
					finished(false)

					continue
				}

				mod := appdb.Mod{
					Section:     string(sectionLocal),
					Name:        strings.Join(fileNames(r.mod), ", "),
					Enabled:     r.mod.Enabled,
					InstalledAt: time.Now(),
					UpdatedAt:   r.mod.Modified,
					Files:       r.files,
				}

				if err := o.installed.AddInstalled(mod); err != nil {
					slog.Warn("a mod from another tool wasn't taken over", "mod", r.mod.Key, "err", err)
					finished(false)

					continue
				}

				o.removeTaken(dir, r.mod)
				finished(true)
			}
		})
	}()
}

// pickFiles returns the files with the ids picked, in GameBanana's order.
func pickFiles(files []banana.File, ids []int) []banana.File {
	var picked []banana.File

	for _, f := range files {
		for _, id := range ids {
			if f.ID == id {
				picked = append(picked, f)

				break
			}
		}
	}

	return picked
}

// removeTaken deletes a mod's files from the addons folder once it's in,
// and takes it out of Deadlock Mod Manager's ledger.
func (o *otherMods) removeTaken(dir string, m deploy.OtherMod) {
	for _, f := range m.Files {
		if err := deploy.RemoveFile(f.Path); err != nil {
			slog.Error("failed to delete a file taken over", "path", f.Path, "err", err)
		}
	}

	if m.Source == deploy.SourceDMM {
		if err := deploy.ForgetInLedger(dir, []string{m.DMMKey}); err != nil {
			slog.Error("failed to update Deadlock Mod Manager's ledger", "err", err)
		}
	}
}

func fileNames(m deploy.OtherMod) []string {
	names := make([]string, len(m.Files))

	for i, f := range m.Files {
		names[i] = f.Name
	}

	return names
}
