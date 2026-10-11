package banana

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestListCategoriesRequest(t *testing.T) {
	tests := []struct {
		name     string
		opts     ListCategoriesOptions
		wantPath string
		want     url.Values
	}{
		{
			name:     "top level",
			opts:     ListCategoriesOptions{},
			wantPath: "/Mod/Categories",
			want: url.Values{
				"_idGameRow":  {"20948"},
				"_sSort":      {"a_to_z"},
				"_bShowEmpty": {"true"},
			},
		},
		{
			name:     "subcategories in another section",
			opts:     ListCategoriesOptions{Section: SectionSounds, ParentID: 6271},
			wantPath: "/Sound/Categories",
			want: url.Values{
				"_idCategoryRow": {"6271"},
				"_sSort":         {"a_to_z"},
				"_bShowEmpty":    {"true"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotQuery url.Values

			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.Query()
				w.Write([]byte("[]"))
			})

			if _, err := c.ListCategories(context.Background(), tt.opts); err != nil {
				t.Fatal(err)
			}

			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}

			if gotQuery.Encode() != tt.want.Encode() {
				t.Errorf("query = %s, want %s", gotQuery.Encode(), tt.want.Encode())
			}
		})
	}
}

func TestListCategoriesDecode(t *testing.T) {
	fixture := readFixture(t, "mod_categories.json")

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	})

	categories, err := c.ListCategories(context.Background(), ListCategoriesOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(categories) != 8 {
		t.Fatalf("len = %d, want 8", len(categories))
	}

	animations := categories[0]
	if animations.ID != 47611 || animations.Name != "Animations" || animations.ItemCount != 18 ||
		animations.CategoryCount != 0 || animations.URL != "https://gamebanana.com/mods/cats/47611" ||
		animations.IconURL != "https://images.gamebanana.com/img/ico/ModCategory/6a64e45dd9812.png" {
		t.Errorf("categories[0] = %+v", animations)
	}

	skins := categories[7]
	if skins.Name != "Skins" || skins.CategoryCount != 44 || skins.IconURL != "" {
		t.Errorf("categories[7] = %+v", skins)
	}
}
