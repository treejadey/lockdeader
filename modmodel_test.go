package main

import (
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

// Five pages of three mods, numbered 1 to 15.
func newSeekServer(t *testing.T) *banana.Client {
	page := []string{"show", "show", "show"}

	return newPagedServer(t, [][]string{page, page, page, page, page})
}

func TestModListSeek(t *testing.T) {
	m := mainthread.Wait2(func() *modListModel {
		return newModListModel(newSeekServer(t), qml.NewQQmlPropertyMap())
	})

	found := make(chan map[int]int, 1)

	// 5 is on page 2, and 14 on page 5, past the 3 pages looked through.
	mainthread.Wait(func() {
		m.FetchNextPage()
		m.Seek([]int{5, 14}, 3, func(rows map[int]int) { found <- rows })
	})

	rows := <-found

	if len(rows) != 1 || rows[5] != 4 {
		t.Errorf("rows = %v, want only mod 5, in row 4", rows)
	}

	if pages := mainthread.Wait2(func() int { return m.nextPage - 1 }); pages != 3 {
		t.Errorf("%d pages loaded, want 3", pages)
	}
}

func TestModListSeekStopsOnceFound(t *testing.T) {
	m := mainthread.Wait2(func() *modListModel {
		return newModListModel(newSeekServer(t), qml.NewQQmlPropertyMap())
	})

	found := make(chan map[int]int, 1)

	mainthread.Wait(func() {
		m.FetchNextPage()
		m.Seek([]int{2}, 3, func(rows map[int]int) { found <- rows })
	})

	if rows := <-found; rows[2] != 1 {
		t.Errorf("rows = %v, want mod 2 in row 1", rows)
	}

	if pages := mainthread.Wait2(func() int { return m.nextPage - 1 }); pages != 1 {
		t.Errorf("%d pages loaded, want only the first", pages)
	}
}

func TestModListSeekGivesUpWhenTheListChanges(t *testing.T) {
	m := mainthread.Wait2(func() *modListModel {
		return newModListModel(newSeekServer(t), qml.NewQQmlPropertyMap())
	})

	found := make(chan map[int]int, 1)

	mainthread.Wait(func() {
		m.FetchNextPage()
		m.Seek([]int{5}, 3, func(rows map[int]int) { found <- rows })
		m.SetCategory(banana.SectionMods, 42)
	})

	if rows := <-found; rows != nil {
		t.Errorf("rows = %v after the category changed, want nil", rows)
	}
}
