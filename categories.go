package main

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

type categoryNode struct {
	Section  banana.Section
	ID       int // 0 for a whole section
	Name     string
	IconURL  string // optional
	Children []categoryNode

	itemCount int
}

// Sounds go last, as one more category with the sound categories under it.
// Every level of subcategories is fetched up front, so the sidebar can open
// them without waiting.
func loadCategories(ctx context.Context, client *banana.Client) ([]categoryNode, error) {
	mods, err := loadSubcategories(ctx, client, banana.SectionMods, 0)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(mods, func(a, b categoryNode) int {
		return cmp.Compare(b.itemCount, a.itemCount)
	})

	sounds, err := loadSubcategories(ctx, client, banana.SectionSounds, 0)
	if err != nil {
		return nil, err
	}

	return append(mods, categoryNode{
		Section:  banana.SectionSounds,
		Name:     "Sounds",
		Children: sounds,
	}), nil
}

// Subcategories come alphabetically. A parentID of 0 fetches the section's
// top-level categories.
func loadSubcategories(ctx context.Context, client *banana.Client, section banana.Section, parentID int) ([]categoryNode, error) {
	entries, err := client.ListCategories(ctx, banana.ListCategoriesOptions{Section: section, ParentID: parentID})
	if err != nil {
		return nil, err
	}

	var nodes []categoryNode
	var parents []int // indexes into nodes of the categories with subcategories

	for _, e := range entries {
		if e.ItemCount == 0 || e.IsObsolete {
			continue
		}

		if e.CategoryCount > 0 {
			parents = append(parents, len(nodes))
		}

		nodes = append(nodes, categoryNode{
			Section:   section,
			ID:        e.ID,
			Name:      e.Name,
			IconURL:   e.IconURL,
			itemCount: e.ItemCount,
		})
	}

	// Fetch every category's subcategories at once.
	var wg sync.WaitGroup
	errs := make([]error, len(parents))

	for i, n := range parents {
		wg.Go(func() {
			nodes[n].Children, errs[i] = loadSubcategories(ctx, client, section, nodes[n].ID)
		})
	}

	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return nodes, nil
}

func categoryList(nodes []categoryNode) *qt.QVariant {
	list := make([]qt.QVariant, 0, len(nodes))

	for _, n := range nodes {
		list = append(list, *qt.NewQVariant20(map[string]qt.QVariant{
			"section":  *qt.NewQVariant14(string(n.Section)),
			"id":       *qt.NewQVariant4(n.ID),
			"name":     *qt.NewQVariant14(n.Name),
			"icon":     *qt.NewQVariant14(n.IconURL),
			"children": *categoryList(n.Children),
		}))
	}

	return qt.NewQVariant43(list)
}

// categoryBrowser backs the category list in the sidebar. Go fills in "tree"
// with the categories (see categoryList), and "loading" and "error" while
// fetching them. QML picks what the mod grid shows by writing {section, id}
// to "selected", where an id of 0 means the whole section.
//
// All methods must be called on the Qt main thread.
type categoryBrowser struct {
	*qml.QQmlPropertyMap

	client   *banana.Client
	mods     *modListModel
	onChange func() // called after switching to a different category; may be nil
	// Called with the category switched to. May be nil.
	onSelected func(section banana.Section, id int)
	loading    bool
}

func newCategoryBrowser(client *banana.Client, mods *modListModel, onChange func()) *categoryBrowser {
	b := &categoryBrowser{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		client:          client,
		mods:            mods,
		onChange:        onChange,
	}

	b.Insert("tree", categoryList(nil))
	b.setStatus(false, nil)
	b.Insert("selected", qt.NewQVariant20(map[string]qt.QVariant{
		"section": *qt.NewQVariant14(string(banana.SectionMods)),
		"id":      *qt.NewQVariant4(0),
	}))

	// Only fires for changes made from QML, not for our own Inserts.
	b.OnValueChanged(func(key string, value *qt.QVariant) {
		if key != "selected" {
			return
		}

		selected := value.ToMap()
		sectionValue, idValue := selected["section"], selected["id"]
		b.Select(banana.Section(sectionValue.ToString()), idValue.ToInt())
	})

	return b
}

func (b *categoryBrowser) Select(section banana.Section, id int) {
	if section != banana.SectionMods && section != banana.SectionSounds {
		slog.Error("unknown section selected", "section", section)

		return
	}

	if !b.mods.SetCategory(section, id) {
		return
	}

	if b.onSelected != nil {
		b.onSelected(section, id)
	}

	if b.onChange != nil {
		b.onChange()
	}
}

// Restore starts the listing in the category the user left off in. An unknown
// section starts it at all mods.
func (b *categoryBrowser) Restore(section banana.Section, id int) {
	if section != banana.SectionMods && section != banana.SectionSounds {
		section, id = banana.SectionMods, 0
	}

	b.Insert("selected", qt.NewQVariant20(map[string]qt.QVariant{
		"section": *qt.NewQVariant14(string(section)),
		"id":      *qt.NewQVariant4(id),
	}))

	// It only reloads the listing if the category is a different one.
	if !b.mods.SetCategory(section, id) {
		b.mods.FetchNextPage()
	}
}

func (b *categoryBrowser) Load() {
	if b.loading {
		return
	}

	b.setStatus(true, nil)

	go func() {
		categories, err := loadCategories(context.Background(), b.client)

		mainthread.Start(func() {
			if err != nil {
				slog.Error("failed to fetch categories", "err", err)
				b.setStatus(false, err)

				return
			}

			b.setStatus(false, nil)
			b.Insert("tree", categoryList(categories))
		})
	}()
}

func (b *categoryBrowser) setStatus(loading bool, err error) {
	b.loading = loading
	b.Insert("loading", qt.NewQVariant8(loading))
	insertError(b.QQmlPropertyMap, err)
}
