package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
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

// newSearchServer lists mods with the given names, whatever is asked for, and
// records the queries. If release isn't nil, each request waits for a value
// from it.
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

// TestModSearchSort sends the picked order with every listing, in any
// category and with a search, and remembers it.
func TestModSearchSort(t *testing.T) {
	var requests []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mainthread.Wait(func() { requests = append(requests, r.URL.Query()) })
		w.Write([]byte(`{"_aMetadata": {"_nRecordCount": 0, "_bIsComplete": true}, "_aRecords": []}`))
	}))
	t.Cleanup(srv.Close)

	client := banana.NewClient()
	client.BaseURL = srv.URL

	var remembered []banana.Sort
	var changes int
	var mods *modListModel

	s := mainthread.Wait2(func() *modSearch {
		mods = newModListModel(client, qml.NewQQmlPropertyMap())
		s := newModSearch(mods, func(int) {}, func() { changes++ })
		s.onSort = func(sort banana.Sort) { remembered = append(remembered, sort) }

		return s
	})

	// Each listing's last request.
	last := func(n int) url.Values {
		t.Helper()

		waitFor(t, "the listing", func() bool { return len(requests) >= n })

		return mainthread.Wait2(func() url.Values { return requests[n-1] })
	}

	mainthread.Wait(func() { s.SetSort(banana.SortMostLiked) })

	if got := last(1).Get("_sSort"); got != "Generic_MostLiked" {
		t.Errorf("_sSort = %q, want Generic_MostLiked", got)
	}

	mainthread.Wait(func() { mods.SetCategory(banana.SectionSounds, 6270) })
	mainthread.Wait(func() { s.SetQuery("abrams") })

	q := last(3)
	if q.Get("_sSort") != "Generic_MostLiked" || q.Get("_aFilters[Generic_Category]") != "6270" ||
		q.Get("_aFilters[Generic_Name]") != "contains,abrams" {
		t.Errorf("request with a category and a search = %v", q)
	}

	// An order the page doesn't offer changes nothing.
	mainthread.Wait(func() { s.SetSort("Generic_Bogus") })

	// GameBanana's own order sends no sort.
	mainthread.Wait(func() { s.SetSort(banana.SortDefault) })

	if q := last(4); q.Has("_sSort") {
		t.Errorf("_sSort = %q for GameBanana's own order, want none", q.Get("_sSort"))
	}

	mainthread.Wait(func() {
		if len(requests) != 4 {
			t.Errorf("%d requests, want 4", len(requests))
		}

		if want := []banana.Sort{banana.SortMostLiked, banana.SortDefault}; !slices.Equal(remembered, want) {
			t.Errorf("remembered %q, want %q", remembered, want)
		}

		if got := s.Value("sort").ToString(); got != "" {
			t.Errorf("sort = %q, want \"\"", got)
		}

		// The sorts, the search, and back.
		if changes != 3 {
			t.Errorf("onChange called %d times, want 3", changes)
		}

		if got := len(s.Value("sorts").ToList()); got != len(modSorts()) {
			t.Errorf("%d sorts offered, want %d", got, len(modSorts()))
		}
	})
}
