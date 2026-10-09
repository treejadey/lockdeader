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
)

var modRoleNames = map[int][]byte{
	roleModID:     []byte("modId"),
	roleName:      []byte("name"),
	roleAuthor:    []byte("author"),
	roleCategory:  []byte("category"),
	roleThumbnail: []byte("thumbnail"),
	roleUpdated:   []byte("updated"),
}

// thumbnailWidth is the preview variant requested from GameBanana. Cards are
// never wider than this, so previews stay sharp without fetching full images.
const thumbnailWidth = 530

// modListModel exposes GameBanana's mod listing to QML views. Pages are fetched
// on demand: views call canFetchMore/fetchMore as they approach the last row.
// All methods must be called on the Qt main thread.
//
// It's a QStandardItemModel rather than a QAbstractListModel subclass because
// miqt v0.14.0's roleNames override frees its return value before Qt reads it.
type modListModel struct {
	*qt.QStandardItemModel

	client *banana.Client
	state  *qml.QQmlPropertyMap // "loading" and "error", for the UI

	byID map[int]banana.Mod // every mod loaded so far

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
		byID:               make(map[int]banana.Mod),
		nextPage:           1,
	}
	m.SetItemRoleNames(modRoleNames)
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

	page := m.nextPage
	go func() {
		result, err := m.client.ListMods(context.Background(), banana.ListModsOptions{Page: page})

		mainthread.Start(func() {
			m.setLoading(false)
			if err != nil {
				slog.Error("failed to fetch mods", "page", page, "err", err)
				m.setError(err)
				return
			}
			slog.Info("fetched mods", "page", page, "count", len(result.Mods), "total", result.Total)
			m.nextPage++
			m.done = result.IsLast
			m.append(result.Mods)
		})
	}()
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

func (m *modListModel) append(mods []banana.Mod) {
	for _, mod := range mods {
		m.byID[mod.ID] = mod

		item := qt.NewQStandardItem()
		item.SetEditable(false)
		item.SetData(qt.NewQVariant4(mod.ID), roleModID)
		item.SetData(qt.NewQVariant14(mod.Name), roleName)
		item.SetData(qt.NewQVariant14(mod.Submitter.Name), roleAuthor)
		item.SetData(qt.NewQVariant14(modCategory(mod.RootCategory, mod.SubCategory)), roleCategory)
		item.SetData(qt.NewQVariant14(previewURL(mod.PreviewMedia)), roleThumbnail)
		// A Unix timestamp; QML formats it relative to now ("3 hours ago").
		item.SetData(qt.NewQVariant6(mod.LastUpdated().Unix()), roleUpdated)
		m.AppendRow([]*qt.QStandardItem{item})
	}
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
