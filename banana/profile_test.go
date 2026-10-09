package banana

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetModProfile(t *testing.T) {
	fixture := readFixture(t, "mod_profile.json")
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write(fixture)
	})

	p, err := c.GetModProfile(context.Background(), 599927)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/Mod/599927/ProfilePage" {
		t.Errorf("path = %q, want /Mod/599927/ProfilePage", gotPath)
	}

	if p.ID != 599927 || p.Name != "Yamato redesign (NEW ICONS and COLORS!)" {
		t.Errorf("profile = %d %q", p.ID, p.Name)
	}
	if !strings.HasPrefix(p.Text, "Redesigned Yamato with a new look<br>") {
		t.Errorf("Text = %q", p.Text)
	}
	if p.Summary != "Yamato NEW model!" {
		t.Errorf("Summary = %q", p.Summary)
	}
	if p.Submitter.Name != "Kirill Senzu" {
		t.Errorf("Submitter = %q", p.Submitter.Name)
	}
	if p.Category.Name != "Yamato" || p.SuperCategory == nil || p.SuperCategory.Name != "Skins" {
		t.Errorf("categories = %q / %v", p.Category.Name, p.SuperCategory)
	}
	if img, ok := p.PreviewMedia.First(); !ok || img.SizedURL(530) != "https://images.gamebanana.com/img/ss/mods/530-90_6988989395e38.jpg" {
		t.Errorf("preview = %v %v", img, ok)
	}

	if len(p.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(p.Files))
	}
	f := p.Files[0]
	if f.ID != 1627082 || f.Name != "yamato_red_3f484.rar" || f.Size != 24670571 ||
		f.DownloadURL != "https://gamebanana.com/dl/1627082" || f.MD5 != "5c87adef48470236adff65cbe693fbd6" {
		t.Errorf("file = %+v", f)
	}
}

func TestGetModProfileAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetModProfile(context.Background(), 1)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want *APIError with 404", err)
	}
}
