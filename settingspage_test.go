package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdata"
	"github.com/treejadey/lockdeader/banana"
)

func TestSettingsPageShowNSFW(t *testing.T) {
	mainthread.Wait(func() {
		var changes []bool
		p := newSettingsPage(appdata.BrowseSettings{ShowNSFW: true}, "/data/Lockdeader",
			func(show bool) { changes = append(changes, show) })

		if !p.Value("showNsfw").ToBool() {
			t.Error("showNsfw = false, want true")
		}

		if got := p.Value("dataDir").ToString(); got != "/data/Lockdeader" {
			t.Errorf("dataDir = %q", got)
		}

		if got := p.Value("dataDirUrl").ToString(); got != "file:///data/Lockdeader" {
			t.Errorf("dataDirUrl = %q", got)
		}

		// As if from QML: Insert doesn't call OnValueChanged.
		p.SetShowNSFW(false)
		p.SetShowNSFW(false)
		p.SetShowNSFW(true)

		if fmt.Sprint(changes) != "[false true]" {
			t.Errorf("changes = %v, want [false true]", changes)
		}

		if !p.Value("showNsfw").ToBool() {
			t.Error("showNsfw = false after showing them again")
		}
	})
}

// newPagedServer serves a listing whose pages hold mods with the given
// visibilities (see banana.Mod.InitialVisibility), numbered from 1 in order.
func newPagedServer(t *testing.T, pages [][]string) *banana.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("_nPage"))

		// Mods are numbered on from those on earlier pages.
		id := 0

		for _, p := range pages[:page-1] {
			id += len(p)
		}

		records := "["

		for i, visibility := range pages[page-1] {
			if i > 0 {
				records += ","
			}

			id++
			records += fmt.Sprintf(`{"_idRow": %d, "_sName": "Mod %d", "_sInitialVisibility": %q}`, id, id, visibility)
		}

		records += "]"
		fmt.Fprintf(w, `{"_aMetadata": {"_bIsComplete": %t}, "_aRecords": %s}`, page == len(pages), records)
	}))
	t.Cleanup(srv.Close)

	client := banana.NewClient()
	client.BaseURL = srv.URL

	return client
}

func TestModListHidesNSFW(t *testing.T) {
	client := newPagedServer(t, [][]string{
		{"show", "warn", "hide"},
		// A page with nothing to list: the next one is fetched anyway.
		{"warn", "hide"},
		{"hide", "show"},
	})

	modIDs := func(m *modListModel) string {
		ids := make([]int, m.RowCount(qt.NewQModelIndex()))

		for row := range ids {
			ids[row] = m.Data(m.Index(row, 0, qt.NewQModelIndex()), roleModID).ToInt()
		}

		return fmt.Sprint(ids)
	}

	m := mainthread.Wait2(func() *modListModel {
		m := newModListModel(client, qml.NewQQmlPropertyMap())
		m.SetHideNSFW(true)

		return m
	})

	waitFor(t, "the first page", func() bool { return m.Loaded() && !m.loading })

	mainthread.Wait(func() {
		if got := modIDs(m); got != "[1]" {
			t.Errorf("rows after the first page = %v, want [1]", got)
		}

		m.FetchNextPage()
	})

	waitFor(t, "the last page", func() bool { return m.done })

	mainthread.Wait(func() {
		if got := modIDs(m); got != "[1 7]" {
			t.Errorf("rows with NSFW mods hidden = %v, want [1 7]", got)
		}

		if m.SetHideNSFW(true) {
			t.Error("hiding NSFW mods again reported a change")
		}

		if !m.SetHideNSFW(false) {
			t.Error("showing NSFW mods reported no change")
		}
	})

	waitFor(t, "the reloaded first page", func() bool { return m.Loaded() && !m.loading })

	mainthread.Wait(func() {
		if got := modIDs(m); got != "[1 2 3]" {
			t.Errorf("rows with NSFW mods shown = %v, want [1 2 3]", got)
		}
	})
}
