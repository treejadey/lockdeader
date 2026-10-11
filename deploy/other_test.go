package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScanOtherMods(t *testing.T) {
	game := newGame(t)
	citadel := filepath.Join(game, "game", "citadel")
	addons := filepath.Join(citadel, "addons")
	addons2 := filepath.Join(citadel, "addons2")
	os.MkdirAll(filepath.Join(addons, "replays"), 0o755)
	os.MkdirAll(addons2, 0o755)

	write := func(dir, name, data string) {
		t.Helper()

		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A ledger as Deadlock Mod Manager writes it, with a mod of two files,
	// a sound, one turned off, one spilled over, and one whose file is gone.
	write(addons, ".dmm.json", `{"version": 3, "mods": {
		"682482": {"enabled": true, "order": null, "currentVpks": ["pak03_dir.vpk", "pak04_dir.vpk"], "disabledVpks": [], "originalVpkNames": ["icons.vpk", "rosia.vpk"]},
		"snd-87131": {"enabled": true, "currentVpks": ["pak01_dir.vpk"], "disabledVpks": [], "originalVpkNames": ["pak50_dir.vpk"]},
		"634732": {"enabled": false, "currentVpks": [], "disabledVpks": ["634732_pak88_dir.vpk"], "originalVpkNames": ["pak88_dir.vpk"]},
		"local-abc": {"enabled": true, "shard": 2, "currentVpks": ["pak01_dir.vpk"], "disabledVpks": [], "originalVpkNames": ["mine.vpk"]},
		"999": {"enabled": true, "currentVpks": ["pak09_dir.vpk"], "disabledVpks": [], "originalVpkNames": ["gone.vpk"]}
	}}`)
	write(addons, "pak01_dir.vpk", "sound")
	write(addons, "pak03_dir.vpk", "icons")
	write(addons, "pak04_dir.vpk", "rosia!")
	write(addons, "634732_pak88_dir.vpk", "idols")
	write(addons, "pak07_dir.vpk", "by hand")
	write(addons, "old_mod.vpk", "by hand, off")
	write(addons, "readme.txt", "not a vpk")
	write(addons2, "pak01_dir.vpk", "local")

	mods, err := ScanOtherMods(game)
	if err != nil {
		t.Fatal(err)
	}

	type want struct {
		key     string
		source  Source
		id      int
		section string
		enabled bool
		files   []string
		names   []string
	}

	wants := []want{
		{"dmm/snd-87131", SourceDMM, 87131, "Sound", true, []string{"addons/pak01_dir.vpk"}, []string{"pak50_dir.vpk"}},
		{"dmm/682482", SourceDMM, 682482, "", true, []string{"addons/pak03_dir.vpk", "addons/pak04_dir.vpk"}, []string{"icons.vpk", "rosia.vpk"}},
		{"addons/addons/pak07_dir.vpk", SourceAddons, 0, "", true, []string{"addons/pak07_dir.vpk"}, []string{"pak07_dir.vpk"}},
		{"dmm/local-abc", SourceDMM, 0, "", true, []string{"addons2/pak01_dir.vpk"}, []string{"mine.vpk"}},
		{"dmm/634732", SourceDMM, 634732, "", false, []string{"addons/634732_pak88_dir.vpk"}, []string{"pak88_dir.vpk"}},
		{"addons/addons/old_mod.vpk", SourceAddons, 0, "", false, []string{"addons/old_mod.vpk"}, []string{"old_mod.vpk"}},
	}

	if len(mods) != len(wants) {
		t.Fatalf("got %d mods, want %d: %+v", len(mods), len(wants), mods)
	}

	for i, w := range wants {
		m := mods[i]

		var files, names []string

		for _, f := range m.Files {
			rel, _ := filepath.Rel(citadel, f.Path)
			files = append(files, filepath.ToSlash(rel))
			names = append(names, f.Name)

			if f.Size == 0 || m.Modified.IsZero() {
				t.Errorf("%s: size %d, modified %v", m.Key, f.Size, m.Modified)
			}
		}

		if m.Key != w.key || m.Source != w.source || m.GameBananaID != w.id || m.Section != w.section ||
			m.Enabled != w.enabled || !equal(files, w.files) || !equal(names, w.names) {
			t.Errorf("mod %d = %+v\nwant %+v", i, m, w)
		}
	}

	// Taking the ledger's mods over forgets them, and the rest is kept as it was.
	if err := ForgetInLedger(game, []string{"682482", "snd-87131", "999"}); err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	data, _ := os.ReadFile(filepath.Join(addons, ".dmm.json"))

	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	var left map[string]map[string]any
	json.Unmarshal(raw["mods"], &left)

	if string(raw["version"]) != "3" || len(left) != 2 || left["634732"] == nil || left["local-abc"]["shard"] != float64(2) {
		t.Errorf("ledger after forgetting: %s", data)
	}

	if err := ForgetInLedger(game, []string{"634732", "local-abc"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(addons, ".dmm.json")); !os.IsNotExist(err) {
		t.Error("an empty ledger was kept")
	}

	if err := ForgetInLedger(game, []string{"x"}); err != nil {
		t.Errorf("forgetting without a ledger: %v", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestScanOtherModsNone(t *testing.T) {
	mods, err := ScanOtherMods(newGame(t))
	if err != nil || len(mods) != 0 {
		t.Errorf("got %v, %v", mods, err)
	}

	// A ledger that can't be read is an error, not a folder of loose files.
	game := newGame(t)
	addons := filepath.Join(game, "game", "citadel", "addons")
	os.MkdirAll(addons, 0o755)
	os.WriteFile(filepath.Join(addons, ".dmm.json"), []byte("{"), 0o644)
	os.WriteFile(filepath.Join(addons, "pak01_dir.vpk"), []byte("x"), 0o644)

	if _, err := ScanOtherMods(game); err == nil {
		t.Error("no error for a broken ledger")
	}
}
