package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/install"
	"github.com/treejadey/lockdeader/vpk"
)

// vpkFile writes a VPK with one file to path.
func vpkFile(t *testing.T, path, name, data string) {
	t.Helper()

	var b bytes.Buffer
	err := vpk.Write(&b, []vpk.Source{{
		Path: name, CRC32: crc32.ChecksumIEEE([]byte(data)), Size: uint32(len(data)),
		Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(data)), nil },
	}})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOtherMods(t *testing.T) {
	// A game folder with a DMM mod of two files (a sound, by an id the
	// ledger doesn't say the section of), a DMM mod that's off, and a
	// loose VPK.
	game := filepath.Join(t.TempDir(), "Deadlock")
	addons := filepath.Join(game, "game", "citadel", "addons")
	os.MkdirAll(addons, 0o755)
	os.WriteFile(filepath.Join(game, "game", "citadel", "gameinfo.gi"), []byte("SearchPaths\n{\n\tGame citadel\n}\n"), 0o644)
	os.WriteFile(filepath.Join(addons, ".dmm.json"), []byte(`{"version": 1, "mods": {
		"87131": {"enabled": true, "currentVpks": ["pak01_dir.vpk", "pak02_dir.vpk"], "disabledVpks": [], "originalVpkNames": ["a.vpk", "b.vpk"]},
		"5": {"enabled": false, "currentVpks": [], "disabledVpks": ["5_off.vpk"], "originalVpkNames": ["off.vpk"]}
	}}`), 0o644)
	vpkFile(t, filepath.Join(addons, "pak01_dir.vpk"), "sounds/a.vsnd_c", "a")
	vpkFile(t, filepath.Join(addons, "pak02_dir.vpk"), "sounds/b.vsnd_c", "b")
	vpkFile(t, filepath.Join(addons, "5_off.vpk"), "sounds/off.vsnd_c", "off")
	vpkFile(t, filepath.Join(addons, "pak09_dir.vpk"), "sounds/hand.vsnd_c", "by hand")

	// GameBanana knows 87131 as a sound (the mod by that id is another
	// game's) with two files, and not 5 at all.
	var srv *httptest.Server
	downloadPath := filepath.Join(t.TempDir(), "download.vpk")
	vpkFile(t, downloadPath, "sounds/real.vsnd_c", "from GameBanana")
	download, _ := os.ReadFile(downloadPath)

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Mod/87131/ProfilePage":
			json.NewEncoder(w).Encode(map[string]any{"_idRow": 87131, "_sName": "Not Deadlock's", "_aGame": map[string]any{"_idRow": 1}})
		case "/Sound/87131/ProfilePage":
			json.NewEncoder(w).Encode(map[string]any{"_idRow": 87131, "_sName": "Freebird Owl", "_aGame": map[string]any{"_idRow": banana.DeadlockGameID},
				"_aSubmitter": map[string]any{"_sName": "Someone"}, "_tsDateAdded": 1700000000,
				"_aFiles": []map[string]any{
					{"_idRow": 11, "_sFile": "other.zip", "_nFilesize": 10, "_sDownloadUrl": srv.URL + "/dl/11"},
					{"_idRow": 12, "_sFile": "b_v2.zip", "_nFilesize": len(download), "_sDownloadUrl": srv.URL + "/dl/12"},
				}})
		case "/dl/12":
			w.Write(download)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"_sErrorCode": "NOT_FOUND"}`)
		}
	}))
	defer srv.Close()

	client := banana.NewClient()
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	db, err := appdb.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	store := install.Store{Dir: t.TempDir()}
	var o *otherMods
	var installed *installedMods
	changes := 0

	mainthread.Wait(func() {
		installed = newInstalledMods(client, db, store)
		installed.onChange = func() { changes++ }
		o = newOtherMods(client, store, installed, func() string { return game })
		o.Scan()
	})

	role := func(row int, role int) *qt.QVariant {
		return o.Model.Item(row).Data(role)
	}

	waitFor(t, "the scan", func() bool { return o.Value("count").ToInt() == 3 && !o.scanning })
	waitFor(t, "the lookup", func() bool { return role(0, roleInstalledName).ToString() == "Freebird Owl" })

	mainthread.Wait(func() {
		if o.Value("dmmCount").ToInt() != 2 {
			t.Errorf("dmmCount = %d", o.Value("dmmCount").ToInt())
		}

		// On first, then the loose file, then the one that's off.
		if got := role(0, roleInstalledKey).ToString(); got != "Sound/87131" {
			t.Errorf("row 0 key = %q", got)
		}

		if got := role(0, roleInstalledAuthor).ToString(); got != "Deadlock Mod Manager" {
			t.Errorf("row 0 author = %q", got)
		}

		if got := role(1, roleInstalledName).ToString(); got != "pak09_dir.vpk" || role(1, roleInstalledAuthor).ToString() != "Addons folder" {
			t.Errorf("row 1 = %q by %q", got, role(1, roleInstalledAuthor).ToString())
		}

		if got := role(2, roleInstalledName).ToString(); got != "off.vpk" || role(2, roleInstalledEnabled).ToBool() {
			t.Errorf("row 2 = %q, enabled %v", got, role(2, roleInstalledEnabled).ToBool())
		}

		if !strings.HasSuffix(o.Value("folderUrl").ToString(), "/addons") {
			t.Errorf("folderUrl = %q", o.Value("folderUrl").ToString())
		}

		// The sound has two files to choose from; b_v2.zip is named like
		// one of its VPKs, so it's the guess.
		choices := o.Value("choices").ToList()
		if len(choices) != 1 {
			t.Fatalf("got %d choices", len(choices))
		}

		choice := choices[0].ToMap()
		keyValue, pickedValue, filesValue := choice["key"], choice["picked"], choice["files"]

		if keyValue.ToString() != "dmm/87131" || pickedValue.ToInt() != 12 || len(filesValue.ToList()) != 2 {
			t.Errorf("choice = key %q, picked %d, %d files", keyValue.ToString(), pickedValue.ToInt(), len(filesValue.ToList()))
		}

		if o.Value("lookingUp").ToBool() {
			t.Error("still looking up")
		}

		o.TakeOver(map[string][]int{"dmm/87131": {12}})
	})

	waitFor(t, "the take-over", func() bool { return !o.takingOver && o.Value("count").ToInt() == 0 && !o.scanning })

	mods, err := db.Mods(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(mods) != 3 {
		t.Fatalf("got %d mods: %+v", len(mods), mods)
	}

	// The sound was downloaded from GameBanana, so it's installed like any
	// other; the loose files are local mods.
	var owl, hand, off appdb.Mod

	for _, m := range mods {
		switch m.Name {
		case "Freebird Owl":
			owl = m
		case "pak09_dir.vpk":
			hand = m
		case "off.vpk":
			off = m
		}
	}

	if owl.Section != "Sound" || owl.GameBananaID != 87131 || owl.Author != "Someone" || !owl.Enabled ||
		len(owl.Files) != 1 || owl.Files[0].GameBananaID != 12 || owl.Files[0].Name != "b_v2.zip" ||
		len(owl.Files[0].VPKs) != 1 || owl.Files[0].VPKs[0].Path != "b_v2.zip" {
		t.Errorf("owl = %+v", owl)
	}

	if hand.Section != "Local" || hand.GameBananaID >= 0 || hand.Name != "pak09_dir.vpk" || !hand.Enabled {
		t.Errorf("hand = %+v", hand)
	}

	if off.Section != "Local" || off.Name != "off.vpk" || off.Enabled || !off.Files[0].VPKs[0].Enabled {
		t.Errorf("off = %+v", off)
	}

	// The files are in the store and gone from addons, with the ledger.
	entries, _ := os.ReadDir(addons)
	if len(entries) != 0 {
		t.Errorf("addons still has %d entries", len(entries))
	}

	// The downloaded VPK and the two local ones; DMM's copies of the sound
	// weren't kept.
	shas, _ := store.VPKs()
	if len(shas) != 3 {
		t.Errorf("store has %d VPKs, want 3", len(shas))
	}

	mainthread.Wait(func() {
		if got := installed.Value("installedCount").ToInt(); got != 3 {
			t.Errorf("installedCount = %d", got)
		}

		if changes != 3 {
			t.Errorf("onChange called %d times, want 3", changes)
		}

		if got := installed.Value("message").ToString(); got != "3 mods were taken over" {
			t.Errorf("message = %q", got)
		}
	})

}
