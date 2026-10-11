package main

import (
	"context"
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

// Roles exposed to QML delegates; their names are in modRoleNames.
const (
	roleModID = int(qt.UserRole) + 1 + iota
	roleName
	roleAuthor
	roleCategory
	roleThumbnail
	roleUpdated
	roleNSFW
	roleAudio
)

var modRoleNames = map[int][]byte{
	roleModID:     []byte("modId"),
	roleName:      []byte("name"),
	roleAuthor:    []byte("author"),
	roleCategory:  []byte("category"),
	roleThumbnail: []byte("thumbnail"),
	roleUpdated:   []byte("updated"),
	roleNSFW:      []byte("nsfw"),
	roleAudio:     []byte("audio"),
}

// thumbnailWidth is the preview variant requested from GameBanana. Cards are
// never wider than this, so previews stay sharp without fetching full images.
const thumbnailWidth = 530

// modListModel exposes GameBanana's mod listing to QML views, for one section
// or category at a time, optionally narrowed down by a search. Pages are
// fetched on demand: views call canFetchMore/fetchMore as they approach the
// last row.
// All methods must be called on the Qt main thread.
//
// It's a QStandardItemModel rather than a QAbstractListModel subclass because
// miqt v0.14.0's roleNames override frees its return value before Qt reads it.
type modListModel struct {
	*qt.QStandardItemModel

	client *banana.Client
	state  *qml.QQmlPropertyMap // "section", "loading" and "error", for the UI

	section    banana.Section
	categoryID int                // 0 for the whole section
	query      string             // only mods with this in their name; "" for all
	hideNSFW   bool               // leaves out mods GameBanana marks as NSFW
	generation int                // goes up with every category or query change
	byID       map[int]banana.Mod // every mod loaded so far in this category
	rows       []int              // the ids of the mods in each row

	// Called after a page has been added to the list. Optional.
	onPage func()

	nextPage int
	loading  bool
	done     bool  // the last page has been loaded
	err      error // stops further fetching until Retry is called
}

func newModListModel(client *banana.Client, state *qml.QQmlPropertyMap) *modListModel {
	m := &modListModel{
		QStandardItemModel: qt.NewQStandardItemModel(),
		client:             client,
		state:              state,
		section:            banana.SectionMods,
		byID:               make(map[int]banana.Mod),
		nextPage:           1,
	}
	m.SetItemRoleNames(modRoleNames)
	m.state.Insert("section", qt.NewQVariant14(string(m.section)))
	m.setLoading(false)
	m.setError(nil)

	m.OnCanFetchMore(func(super func(parent *qt.QModelIndex) bool, parent *qt.QModelIndex) bool {
		return !parent.IsValid() && m.canFetch()
	})
	m.OnFetchMore(func(super func(parent *qt.QModelIndex), parent *qt.QModelIndex) {
		if !parent.IsValid() {
			m.FetchNextPage()
		}
	})

	return m
}

func (m *modListModel) canFetch() bool {
	return !m.loading && !m.done && m.err == nil
}

// FetchNextPage starts loading the next page in the background, unless a page
// is already loading or there is nothing left to load.
func (m *modListModel) FetchNextPage() {
	if !m.canFetch() {
		return
	}
	m.setLoading(true)

	generation := m.generation
	opts := banana.ListModsOptions{Section: m.section, CategoryID: m.categoryID, Name: m.query, Page: m.nextPage}
	go func() {
		result, err := m.client.ListMods(context.Background(), opts)

		mainthread.Start(func() {
			if generation != m.generation {
				return // the category or query changed in the meantime
			}
			m.setLoading(false)
			if err != nil {
				slog.Error("failed to fetch mods", "section", opts.Section, "category", opts.CategoryID,
					"query", opts.Name, "page", opts.Page, "err", err)
				m.setError(err)
				return
			}
			slog.Info("fetched mods", "section", opts.Section, "category", opts.CategoryID,
				"query", opts.Name, "page", opts.Page, "count", len(result.Mods), "total", result.Total)
			m.nextPage++
			m.done = result.IsLast
			if m.append(result.Mods) == 0 && !m.done {
				// Every mod on the page was left out. The view won't ask for
				// more when nothing was added, so ask for it here.
				m.FetchNextPage()
				return
			}
			if m.onPage != nil {
				m.onPage()
			}
		})
	}()
}

// SetCategory replaces the list with the mods in a category of section, or in
// the whole section if id is 0, starting again from the first page.
func (m *modListModel) SetCategory(section banana.Section, id int) (changed bool) {
	if section == m.section && id == m.categoryID {
		return false
	}
	m.section = section
	m.categoryID = id
	m.state.Insert("section", qt.NewQVariant14(string(section)))
	m.reload()
	return true
}

// SetQuery replaces the list with the mods whose name contains query,
// ignoring case, in the same category; "" lists them all again.
func (m *modListModel) SetQuery(query string) (changed bool) {
	query = normalizeQuery(query)
	if query == m.query {
		return false
	}
	m.query = query
	m.reload()
	return true
}

// SetHideNSFW leaves mods GameBanana marks as NSFW out of the list, or lists
// them again, starting again from the first page.
func (m *modListModel) SetHideNSFW(hide bool) (changed bool) {
	if hide == m.hideNSFW {
		return false
	}
	m.hideNSFW = hide
	m.reload()
	return true
}

// Query returns the search the list is narrowed down by, if any.
func (m *modListModel) Query() string {
	return m.query
}

// Generation identifies the current listing: it changes whenever the list is
// emptied for another category or query.
func (m *modListModel) Generation() int {
	return m.generation
}

// Loaded reports whether at least one mod of the current listing has
// arrived, or there's nothing to load.
func (m *modListModel) Loaded() bool {
	return len(m.rows) > 0 || m.done
}

// reload empties the list and starts again from the first page.
func (m *modListModel) reload() {
	m.generation++
	m.SetRowCount(0)
	clear(m.byID)
	m.rows = m.rows[:0]
	m.nextPage = 1
	m.done = false
	m.setLoading(false)
	m.setError(nil)
	m.FetchNextPage()
}

// Section returns the section the listed mods are from.
func (m *modListModel) Section() banana.Section {
	return m.section
}

// Lookup returns a mod that has already been loaded into the list.
func (m *modListModel) Lookup(id int) (banana.Mod, bool) {
	mod, ok := m.byID[id]
	return mod, ok
}

// Retry clears a previous fetch error and tries the failed page again.
func (m *modListModel) Retry() {
	m.setError(nil)
	m.FetchNextPage()
}

func (m *modListModel) setLoading(loading bool) {
	m.loading = loading
	m.state.Insert("loading", qt.NewQVariant8(loading))
}

func (m *modListModel) setError(err error) {
	m.err = err
	var msg string
	if err != nil {
		msg = err.Error()
	}
	m.state.Insert("error", qt.NewQVariant14(msg))
}

// append adds mods to the end of the list, except those left out, and
// returns how many it added.
func (m *modListModel) append(mods []banana.Mod) (added int) {
	for _, mod := range mods {
		if m.hideNSFW && mod.IsNSFW() {
			continue
		}
		added++
		m.byID[mod.ID] = mod
		m.rows = append(m.rows, mod.ID)

		item := qt.NewQStandardItem()
		item.SetEditable(false)
		item.SetData(qt.NewQVariant4(mod.ID), roleModID)
		item.SetData(qt.NewQVariant14(mod.Name), roleName)
		item.SetData(qt.NewQVariant14(mod.Submitter.Name), roleAuthor)
		item.SetData(qt.NewQVariant14(modCategory(mod.RootCategory, mod.SubCategory)), roleCategory)
		item.SetData(qt.NewQVariant14(previewURL(mod.PreviewMedia)), roleThumbnail)
		// A Unix timestamp; QML formats it relative to now ("3 hours ago").
		item.SetData(qt.NewQVariant6(mod.LastUpdated().Unix()), roleUpdated)
		item.SetData(qt.NewQVariant8(mod.IsNSFW()), roleNSFW)
		item.SetData(qt.NewQVariant14(mod.PreviewMedia.Metadata.AudioURL), roleAudio)
		m.AppendRow([]*qt.QStandardItem{item})
	}
	return added
}

// modCategory formats a mod's category for display, e.g. "Skins - Vindicta".
func modCategory(category banana.Category, sub *banana.Category) string {
	if sub == nil {
		return category.Name
	}
	return category.Name + " - " + sub.Name
}

// previewURL returns the URL of a mod's main preview image, or "" if it has none.
func previewURL(media banana.PreviewMedia) string {
	img, ok := media.First()
	if !ok {
		return ""
	}
	return img.SizedURL(thumbnailWidth)
}
