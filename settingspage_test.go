package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		p := newSettingsPage(appdata.BrowseSettings{ShowNSFW: true}, appdata.GameSettings{}, "/data/Lockdeader",
			func(show bool) { changes = append(changes, show) }, nil)

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

func TestSettingsPageLoadOtherMods(t *testing.T) {
	mainthread.Wait(func() {
		var changes []appdata.GameSettings
		p := newSettingsPage(appdata.BrowseSettings{}, appdata.GameSettings{LoadOtherMods: true}, "/data",
			nil, func(s appdata.GameSettings) { changes = append(changes, s) })

		if !p.Value("loadOtherMods").ToBool() {
			t.Error("loadOtherMods = false, want true")
		}

		p.SetLoadOtherMods(false)
		p.SetLoadOtherMods(false)
		p.SetLoadOtherMods(true)

		if len(changes) != 2 || changes[0].LoadOtherMods || !changes[1].LoadOtherMods {
			t.Errorf("changes = %+v", changes)
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

// fakeDeadlock makes a folder that game.Check takes as Deadlock's.
func fakeDeadlock(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "Deadlock")
	path := filepath.Join(dir, "game", "citadel", "gameinfo.gi")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestSettingsPageGameFolder(t *testing.T) {
	deadlock := fakeDeadlock(t)
	other := t.TempDir()

	mainthread.Wait(func() {
		var saved []appdata.GameSettings
		p := newSettingsPage(appdata.BrowseSettings{}, appdata.GameSettings{}, "/data", nil,
			func(s appdata.GameSettings) { saved = append(saved, s) })

		if got := p.Value("gameFolder").ToString(); got != "" {
			t.Errorf("gameFolder = %q, want none", got)
		}

		// Not Deadlock's: nothing changes, and the error says so.
		p.ChooseGameFolder(other)

		if p.Value("gameFolder").ToString() != "" || p.Value("gameFolderError").ToString() == "" || len(saved) != 0 {
			t.Errorf("after choosing another folder: gameFolder = %q, gameFolderError = %q, saved %v",
				p.Value("gameFolder").ToString(), p.Value("gameFolderError").ToString(), saved)
		}

		// A folder inside it counts as the game's own, and clears the error.
		p.ChooseGameFolder(filepath.Join(deadlock, "game", "citadel"))

		if got := p.Value("gameFolder").ToString(); got != deadlock {
			t.Errorf("gameFolder = %q, want %q", got, deadlock)
		}

		if p.Value("gameFolderError").ToString() != "" {
			t.Error("the error is still shown after choosing Deadlock's folder")
		}

		if len(saved) != 1 || saved[0].Path != deadlock {
			t.Errorf("saved %v, want the game folder once", saved)
		}

		// Picking something wrong after doesn't lose the right one.
		p.ChooseGameFolder(other)

		if got := p.Value("gameFolder").ToString(); got != deadlock {
			t.Errorf("gameFolder = %q after a wrong pick, want %q kept", got, deadlock)
		}

		p.SetLaunchOptions("-novid")
		p.SetLaunchOptions("-novid")

		if len(saved) != 2 || saved[1].LaunchOptions != "-novid" || saved[1].Path != deadlock {
			t.Errorf("saved %v, want the launch options once, with the folder", saved)
		}
	})
}

func TestSettingsPageFindGame(t *testing.T) {
	deadlock := fakeDeadlock(t)
	found := make(chan appdata.GameSettings, 1)

	mainthread.Wait(func() {
		p := newSettingsPage(appdata.BrowseSettings{}, appdata.GameSettings{}, "/data", nil,
			func(s appdata.GameSettings) { found <- s })
		p.find = func() string { return deadlock }
		p.FindGame()
	})

	if s := <-found; s.Path != deadlock {
		t.Errorf("found %q, want %q", s.Path, deadlock)
	}

	// With a folder already, it isn't looked for.
	mainthread.Wait(func() {
		p := newSettingsPage(appdata.BrowseSettings{}, appdata.GameSettings{Path: deadlock}, "/data", nil, nil)
		p.find = func() string {
			t.Error("looked for Deadlock with a folder already set")

			return ""
		}
		p.FindGame()
	})
}
