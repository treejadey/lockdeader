package main

import (
	"cmp"
	"strings"
	"unicode"
	"unicode/utf8"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"
)

// modSearch backs the search field on the mods page. QML writes the search
// to "query", and Go narrows the listing down to the mods whose names contain
// it. Writing true to "pick" opens the loaded mod that matches the search best
// (see matchRank), waiting for the results first if they're still on their
// way. Go then puts the mod's row in "pickedRow" and bumps "pickId", so the
// view can scroll to it.
//
// All methods must be called on the Qt main thread.
type modSearch struct {
	*qml.QQmlPropertyMap

	mods     *modListModel
	open     func(id int) // opens a mod in the details panel
	onChange func()       // may be nil

	pendingPick int // the listing generation to pick from once it loads; -1 for none
	picks       int
}

func newModSearch(mods *modListModel, open func(id int), onChange func()) *modSearch {
	s := &modSearch{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		mods:            mods,
		open:            open,
		onChange:        onChange,
		pendingPick:     -1,
	}

	s.Insert("query", qt.NewQVariant14(""))
	s.Insert("pick", qt.NewQVariant8(false))
	s.Insert("pickedRow", qt.NewQVariant4(-1))
	s.Insert("pickId", qt.NewQVariant4(0))

	mods.onPage = func() {
		if s.pendingPick == mods.Generation() {
			s.pendingPick = -1
			s.pickNow()
		}
	}

	// Only fires for changes made from QML, not for our own Inserts.
	s.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "query":
			s.SetQuery(value.ToString())
		case "pick":
			if value.ToBool() {
				// Clear the key, so picking again is still a change.
				s.Insert("pick", qt.NewQVariant8(false))
				s.Pick()
			}
		}
	})

	return s
}

func (s *modSearch) SetQuery(query string) {
	if s.mods.SetQuery(query) && s.onChange != nil {
		s.onChange()
	}
}

func (s *modSearch) Pick() {
	if s.mods.Query() == "" {
		return
	}

	if !s.mods.Loaded() {
		s.pendingPick = s.mods.Generation()

		return
	}

	s.pickNow()
}

func (s *modSearch) pickNow() {
	row, ok := s.mods.BestMatch()
	if !ok {
		return
	}

	s.open(s.mods.rows[row])

	s.picks++
	s.Insert("pickedRow", qt.NewQVariant4(row))
	s.Insert("pickId", qt.NewQVariant4(s.picks))
}

// BestMatch goes by matchRank, then the shortest name, as the one with the
// least besides the search in it, then the newest.
func (m *modListModel) BestMatch() (row int, ok bool) {
	if m.query == "" || len(m.rows) == 0 {
		return 0, false
	}

	query := strings.ToLower(m.query)
	best, bestRank, bestLength := -1, 0, 0

	for i, id := range m.rows {
		name := strings.ToLower(normalizeQuery(m.byID[id].Name))
		rank, length := matchRank(name, query), utf8.RuneCountInString(name)
		if best < 0 || cmp.Or(cmp.Compare(rank, bestRank), cmp.Compare(length, bestLength)) < 0 {
			best, bestRank, bestLength = i, rank, length
		}
	}

	return best, true
}

// matchRank rates how well name matches query, both in lower case; lower is
// better:
//
//	0: the name is the query
//	1: the name starts with the query
//	2: a word in the name starts with the query
//	3: the query is somewhere in the name
//	4: every word of the query is somewhere in the name
//	5: none of these, e.g. the server matched it some other way
func matchRank(name, query string) int {
	switch {
	case name == query:
		return 0
	case strings.HasPrefix(name, query):
		return 1
	}

	// Look at every place the query appears, for one that starts a word.
	contains := false

	for i := 0; ; {
		j := strings.Index(name[i:], query)
		if j < 0 {
			break
		}

		contains = true
		i += j

		before, _ := utf8.DecodeLastRuneInString(name[:i])
		if !unicode.IsLetter(before) && !unicode.IsDigit(before) {
			return 2
		}

		_, size := utf8.DecodeRuneInString(name[i:])
		if size == 0 {
			break // an empty query, at the end of the name
		}

		i += size
	}

	if contains {
		return 3
	}

	for _, word := range strings.Fields(query) {
		if !strings.Contains(name, word) {
			return 5
		}
	}

	return 4
}

func normalizeQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
