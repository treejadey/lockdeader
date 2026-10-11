package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

func TestLoadCategories(t *testing.T) {
	// Category listings by section and parent ("" for the top level).
	responses := map[string]string{
		"Mod/": `[
			{"_idRow": 1, "_sName": "Animations", "_nItemCount": 18},
			{"_idRow": 2, "_sName": "Empty", "_nItemCount": 0},
			{"_idRow": 3, "_sName": "Old", "_nItemCount": 5, "_bIsObsolete": true},
			{"_idRow": 4, "_sName": "Skins", "_nItemCount": 3080, "_nCategoryCount": 2,
			 "_sIconUrl": "https://images.gamebanana.com/skins.png"}
		]`,
		"Mod/4": `[
			{"_idRow": 40, "_sName": "Abrams", "_nItemCount": 93},
			{"_idRow": 41, "_sName": "Rat King", "_nItemCount": 111, "_nCategoryCount": 1}
		]`,
		"Mod/41": `[
			{"_idRow": 410, "_sName": "RatKing Flag", "_nItemCount": 90}
		]`,
		"Sound/": `[
			{"_idRow": 6270, "_sName": "Abilities", "_nItemCount": 1858}
		]`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		section := r.URL.Path[1 : len(r.URL.Path)-len("/Categories")]
		body, ok := responses[section+"/"+r.URL.Query().Get("_idCategoryRow")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	client := banana.NewClient()
	client.BaseURL = srv.URL

	got, err := loadCategories(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}

	mod := banana.SectionMods
	want := []categoryNode{
		{Section: mod, ID: 4, Name: "Skins", IconURL: "https://images.gamebanana.com/skins.png", itemCount: 3080,
			Children: []categoryNode{
				{Section: mod, ID: 40, Name: "Abrams", itemCount: 93},
				{Section: mod, ID: 41, Name: "Rat King", itemCount: 111, Children: []categoryNode{
					{Section: mod, ID: 410, Name: "RatKing Flag", itemCount: 90},
				}},
			}},
		{Section: mod, ID: 1, Name: "Animations", itemCount: 18},
		{Section: banana.SectionSounds, Name: "Sounds", Children: []categoryNode{
			{Section: banana.SectionSounds, ID: 6270, Name: "Abilities", itemCount: 1858},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestLoadCategoriesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("_idCategoryRow") {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`[{"_idRow": 4, "_sName": "Skins", "_nItemCount": 1, "_nCategoryCount": 1}]`))
	}))
	t.Cleanup(srv.Close)
	client := banana.NewClient()
	client.BaseURL = srv.URL

	if _, err := loadCategories(context.Background(), client); err == nil {
		t.Error("err = nil, want the subcategory request's error")
	}
}

func TestCategoryBrowserSelect(t *testing.T) {
	// Every listing is empty; only the category switching matters here.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"_aMetadata": {"_nRecordCount": 0, "_bIsComplete": true}, "_aRecords": []}`))
	}))
	t.Cleanup(srv.Close)
	client := banana.NewClient()
	client.BaseURL = srv.URL

	var changes int
	b := mainthread.Wait2(func() *categoryBrowser {
		mods := newModListModel(client, qml.NewQQmlPropertyMap())
		return newCategoryBrowser(client, mods, func() { changes++ })
	})
	selectCategory := func(section banana.Section, id int) int {
		return mainthread.Wait2(func() int {
			b.Select(section, id)
			return changes
		})
	}

	if got := selectCategory(banana.SectionMods, 4); got != 1 {
		t.Errorf("after switching to a category: onChange called %d times, want 1", got)
	}
	if got := selectCategory(banana.SectionMods, 4); got != 1 {
		t.Errorf("after selecting the same category again: onChange called %d times, want still 1", got)
	}
	if got := selectCategory(banana.SectionSounds, 0); got != 2 {
		t.Errorf("after switching to another section: onChange called %d times, want 2", got)
	}
	if got := selectCategory("Bogus", 1); got != 2 {
		t.Errorf("after selecting an unknown section: onChange called %d times, want still 2", got)
	}
}
