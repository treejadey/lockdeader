package banana

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// newTestClient starts a server running handler and returns a Client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient()
	c.BaseURL = srv.URL
	c.HTTPClient = srv.Client()
	return c
}

func TestListModsRequest(t *testing.T) {
	tests := []struct {
		name string
		opts ListModsOptions
		want url.Values
	}{
		{
			name: "defaults",
			opts: ListModsOptions{},
			want: url.Values{
				"_aFilters[Generic_Game]": {"20948"},
				"_nPage":                  {"1"},
				"_nPerpage":               {"50"},
			},
		},
		{
			name: "explicit",
			opts: ListModsOptions{GameID: 123, Page: 4, PerPage: 10, Sort: SortMostLiked},
			want: url.Values{
				"_aFilters[Generic_Game]": {"123"},
				"_nPage":                  {"4"},
				"_nPerpage":               {"10"},
				"_sSort":                  {"Generic_MostLiked"},
			},
		},
		{
			name: "per page clamped",
			opts: ListModsOptions{PerPage: 500},
			want: url.Values{
				"_aFilters[Generic_Game]": {"20948"},
				"_nPage":                  {"1"},
				"_nPerpage":               {"50"},
			},
		},
	}

	fixture := readFixture(t, "mod_index.json")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotQuery url.Values
			var gotUA string
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.Query()
				gotUA = r.UserAgent()
				w.Write(fixture)
			})

			if _, err := c.ListMods(context.Background(), tt.opts); err != nil {
				t.Fatal(err)
			}
			if gotPath != "/Mod/Index" {
				t.Errorf("path = %q, want /Mod/Index", gotPath)
			}
			if gotUA != defaultUserAgent {
				t.Errorf("User-Agent = %q, want %q", gotUA, defaultUserAgent)
			}
			if gotQuery.Encode() != tt.want.Encode() {
				t.Errorf("query = %s, want %s", gotQuery.Encode(), tt.want.Encode())
			}
		})
	}
}

func TestListModsDecode(t *testing.T) {
	fixture := readFixture(t, "mod_index.json")
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	})

	page, err := c.ListMods(context.Background(), ListModsOptions{PerPage: 2})
	if err != nil {
		t.Fatal(err)
	}

	if page.Total != 3 || page.PerPage != 2 || page.Page != 1 || page.IsLast {
		t.Errorf("page meta = {Total:%d PerPage:%d Page:%d IsLast:%v}, want {3 2 1 false}",
			page.Total, page.PerPage, page.Page, page.IsLast)
	}
	if got := page.TotalPages(); got != 2 {
		t.Errorf("TotalPages() = %d, want 2", got)
	}
	if len(page.Mods) != 2 {
		t.Fatalf("len(Mods) = %d, want 2", len(page.Mods))
	}

	first := page.Mods[0]
	if first.ID != 725782 || first.Name != "СЕРЕГА ПИРАТ" {
		t.Errorf("first mod = %d %q", first.ID, first.Name)
	}
	if first.Submitter.Name != "artemka talant" || first.Game.ID != DeadlockGameID {
		t.Errorf("first mod submitter/game = %q/%d", first.Submitter.Name, first.Game.ID)
	}
	if first.RootCategory.Name != "Animations" || first.SubCategory != nil {
		t.Errorf("first mod categories = %q/%v", first.RootCategory.Name, first.SubCategory)
	}
	if !first.UpdatedAt().IsZero() {
		t.Errorf("first mod UpdatedAt = %v, want zero", first.UpdatedAt())
	}
	if !first.LastUpdated().Equal(first.AddedAt()) {
		t.Errorf("first mod LastUpdated = %v, want its added date %v", first.LastUpdated(), first.AddedAt())
	}
	img, ok := first.Thumbnail()
	if !ok {
		t.Fatal("first mod has no thumbnail")
	}
	if got, want := img.SizedURL(220), "https://images.gamebanana.com/img/ss/mods/220-90_6ac8e5dd3af9f.jpg"; got != want {
		t.Errorf("SizedURL(220) = %q, want %q", got, want)
	}

	second := page.Mods[1]
	if second.SubCategory == nil || second.SubCategory.Name != "Rem" {
		t.Errorf("second mod subcategory = %v", second.SubCategory)
	}
	if second.LikeCount != 12 || second.ViewCount != 340 || second.PostCount != 2 {
		t.Errorf("second mod counts = %d/%d/%d", second.LikeCount, second.ViewCount, second.PostCount)
	}
	if len(second.Tags) != 2 || second.Tags[1] != "Source 2" {
		t.Errorf("second mod tags = %v", second.Tags)
	}
	if second.UpdatedAt().Unix() != 1791560000 {
		t.Errorf("second mod UpdatedAt = %v", second.UpdatedAt())
	}
	if second.LastUpdated().Unix() != 1791560000 {
		t.Errorf("second mod LastUpdated = %v, want its update date", second.LastUpdated())
	}
	// Only a 100px variant exists, so asking for 530 should fall back to it.
	img, _ = second.Thumbnail()
	if img.Caption != "Rem, very happy" {
		t.Errorf("Caption = %q", img.Caption)
	}
	if got, want := img.SizedURL(530), "https://images.gamebanana.com/img/ss/mods/100-90_6ac8e63bce9e7.jpg"; got != want {
		t.Errorf("SizedURL(530) = %q, want %q", got, want)
	}
}

func TestListModsAPIError(t *testing.T) {
	fixture := readFixture(t, "error_perpage.json")
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(fixture)
	})

	_, err := c.ListMods(context.Background(), ListModsOptions{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Code != "INPUT_ERRORS" {
		t.Errorf("APIError = %d %q", apiErr.StatusCode, apiErr.Code)
	}
	if apiErr.Fields["_nPerpage"].Code != "INVALID_PERPAGE" {
		t.Errorf("Fields = %v", apiErr.Fields)
	}
	if !strings.Contains(err.Error(), "cannot exceed 50") {
		t.Errorf("Error() = %q, want field message", err.Error())
	}
}

func TestAllMods(t *testing.T) {
	var requested []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("_nPage")
		requested = append(requested, page)
		switch page {
		case "1":
			fmt.Fprint(w, `{"_aMetadata":{"_nRecordCount":3,"_bIsComplete":false,"_nPerpage":2},
				"_aRecords":[{"_idRow":1},{"_idRow":2}]}`)
		case "2":
			fmt.Fprint(w, `{"_aMetadata":{"_nRecordCount":3,"_bIsComplete":true,"_nPerpage":2},
				"_aRecords":[{"_idRow":3}]}`)
		default:
			t.Errorf("unexpected request for page %s", page)
			fmt.Fprint(w, `{"_aMetadata":{},"_aRecords":[]}`)
		}
	})

	var ids []int
	for m, err := range c.AllMods(context.Background(), ListModsOptions{PerPage: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, m.ID)
	}

	if fmt.Sprint(ids) != "[1 2 3]" {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}
	if fmt.Sprint(requested) != "[1 2]" {
		t.Errorf("requested pages = %v, want [1 2]", requested)
	}
}

func TestAllModsError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	var count int
	for _, err := range c.AllMods(context.Background(), ListModsOptions{}) {
		count++
		if err == nil {
			t.Fatal("expected error")
		}
	}
	if count != 1 {
		t.Errorf("yielded %d times, want 1", count)
	}
}
