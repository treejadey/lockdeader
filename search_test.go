package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

func TestMatchRank(t *testing.T) {
	tests := []struct {
		name, query string
		want        int
	}{
		{"green abrams", "green abrams", 0},
		{"abrams burning portrait", "abrams", 1},
		{"green abrams", "abrams", 2},
		{"duke bowser (abrams)", "abrams", 2},
		{"superabrams", "abrams", 3},
		{"superabrams, abrams again", "abrams", 2},
		{"abrams but green", "green abrams", 4},
		{"ábrams", "abrams", 5},
	}
	for _, tt := range tests {
		if got := matchRank(tt.name, tt.query); got != tt.want {
			t.Errorf("matchRank(%q, %q) = %d, want %d", tt.name, tt.query, got, tt.want)
		}
	}
}

// newSearchServer serves a listing of mods with the given names, whatever is
// asked for, and returns the queries it was asked for. Requests wait for
// release to receive a value, if it isn't nil.
func newSearchServer(t *testing.T, names []string, release chan struct{}) (*banana.Client, *[]string) {
	t.Helper()
	var records []string
	for i, name := range names {
		records = append(records, fmt.Sprintf(`{"_idRow": %d, "_sName": %q}`, i+1, name))
	}
	body := fmt.Sprintf(`{"_aMetadata": {"_nRecordCount": %d, "_bIsComplete": true}, "_aRecords": [%s]}`,
		len(names), strings.Join(records, ","))

	queries := new([]string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if release != nil {
			<-release
		}
		mainthread.Wait(func() { *queries = append(*queries, r.URL.Query().Get("_aFilters[Generic_Name]")) })
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	client := banana.NewClient()
	client.BaseURL = srv.URL
	return client, queries
}

func TestModSearchPick(t *testing.T) {
	release := make(chan struct{}, 10)
	client, queries := newSearchServer(t, []string{
		"Green Abrams FOR JILL'S DEADLOCK MOD",
		"Superabrams",
		"Green Abrams",
		"GREEN ABRAMS GLOAT ICON",
	}, release)

	var opened []int
	var changes int
	s := mainthread.Wait2(func() *modSearch {
		mods := newModListModel(client, qml.NewQQmlPropertyMap())
		return newModSearch(mods, func(id int) { opened = append(opened, id) }, func() { changes++ })
	})

	// Picking while the results are still on their way waits for them.
	mainthread.Wait(func() {
		s.SetQuery("  green   abrams ")
		s.Pick()
	})
	if got := mainthread.Wait2(func() int { return len(opened) }); got != 0 {
		t.Fatalf("opened %d mods before the results arrived", got)
	}
	release <- struct{}{}
	waitFor(t, "the pick", func() bool { return len(opened) > 0 })

	mainthread.Wait(func() {
		if want := []string{"contains,green abrams"}; fmt.Sprint(*queries) != fmt.Sprint(want) {
			t.Errorf("queries = %q, want %q", *queries, want)
		}
		if changes != 1 {
			t.Errorf("onChange called %d times, want 1", changes)
		}
		// Every name starts with the query but the second; the shortest wins.
		if len(opened) != 1 || opened[0] != 3 {
			t.Errorf("opened %v, want [3]", opened)
		}
		if got := s.Value("pickedRow").ToInt(); got != 2 {
			t.Errorf("pickedRow = %d, want 2", got)
		}
		if got := s.Value("pickId").ToInt(); got != 1 {
			t.Errorf("pickId = %d, want 1", got)
		}

		// Once the results are in, picking is immediate.
		s.Pick()
		if len(opened) != 2 || s.Value("pickId").ToInt() != 2 {
			t.Errorf("picking again: opened %v, pickId %d", opened, s.Value("pickId").ToInt())
		}

		// The same search, written differently, changes nothing.
		s.SetQuery("green abrams")
		if changes != 1 {
			t.Errorf("onChange called %d times after the same search, want 1", changes)
		}

		// Without a search, there's nothing to pick.
		s.SetQuery("")
		s.Pick()
	})
	release <- struct{}{}
	mainthread.Wait(func() {
		if len(opened) != 2 {
			t.Errorf("picking without a search opened %v", opened)
		}
	})
}
