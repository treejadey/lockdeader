package deploy

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gameWith(t *testing.T, data []byte) string {
	t.Helper()

	game := newGame(t)

	if err := os.WriteFile(GameInfoPath(game), data, 0o644); err != nil {
		t.Fatal(err)
	}

	return game
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	return readFile(t, filepath.Join("testdata", name))
}

func state(t *testing.T, game string) State {
	t.Helper()

	s, err := ReadGameInfo(game)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func enable(t *testing.T, game string, addons bool) bool {
	t.Helper()

	changed, err := EnableMods(game, addons, "")
	if err != nil {
		t.Fatal(err)
	}

	return changed
}

func disable(t *testing.T, game string) bool {
	t.Helper()

	changed, err := DisableMods(game, "")
	if err != nil {
		t.Fatal(err)
	}

	return changed
}

func TestFixtureIsRealShaped(t *testing.T) {
	data := fixture(t, "gameinfo.gi")

	if !bytes.Contains(data, []byte("\r\n")) || !bytes.Contains(data, []byte("\t\t\tGame\t\t\t\tcitadel\r\n")) {
		t.Error("the fixture isn't the stock file: CRLF, tab-indented")
	}
}

func TestReadGameInfoStock(t *testing.T) {
	if s := state(t, newGame(t)); s != (State{}) {
		t.Errorf("State = %+v", s)
	}
}

func TestEnableMods(t *testing.T) {
	game := newGame(t)
	original := readFile(t, GameInfoPath(game))

	if !enable(t, game, false) {
		t.Fatal("not changed")
	}

	data := readFile(t, GameInfoPath(game))
	want := "\t\t\tGame\t\t\t\tcitadel/lockdeader\t// Lockdeader\r\n" +
		"\t\t\tMod\t\t\t\tcitadel\t// Lockdeader\r\n" +
		"\t\t\tWrite\t\t\t\tcitadel\t// Lockdeader\r\n" +
		"\t\t\tGame\t\t\t\tcitadel\r\n"

	if !bytes.Contains(data, []byte(want)) {
		t.Errorf("our lines aren't directly above Game citadel, in its style:\n%s", data)
	}

	if bytes.Contains(bytes.ReplaceAll(data, []byte("\r\n"), nil), []byte("\n")) {
		t.Error("a line ending changed")
	}

	if s := state(t, game); s != (State{Enabled: true}) {
		t.Errorf("State = %+v", s)
	}

	if enable(t, game, false) {
		t.Error("enabling again changed the file")
	}

	if !bytes.Equal(readFile(t, GameInfoPath(game)), data) {
		t.Error("enabling again rewrote the file")
	}

	if !disable(t, game) {
		t.Error("disabling didn't change the file")
	}

	if !bytes.Equal(readFile(t, GameInfoPath(game)), original) {
		t.Error("disabling didn't restore the file exactly")
	}

	if disable(t, game) {
		t.Error("disabling again changed the file")
	}
}

func TestEnableModsAddons(t *testing.T) {
	game := newGame(t)
	original := readFile(t, GameInfoPath(game))

	enable(t, game, true)

	data := string(readFile(t, GameInfoPath(game)))
	ours := "\t\t\tGame\t\t\t\tcitadel/lockdeader\t// Lockdeader\r\n\t\t\tGame\t\t\t\tcitadel/addons\t// Lockdeader\r\n\t\t\tMod\t\t\t\tcitadel\t// Lockdeader\r\n"

	if !strings.Contains(data, ours) {
		t.Errorf("addons line isn't after ours:\n%s", data)
	}

	if s := state(t, game); s != (State{Enabled: true, Addons: true}) {
		t.Errorf("State = %+v", s)
	}

	if !enable(t, game, false) {
		t.Error("turning addons off didn't change the file")
	}

	if s := state(t, game); s != (State{Enabled: true}) {
		t.Errorf("State = %+v", s)
	}

	disable(t, game)

	if !bytes.Equal(readFile(t, GameInfoPath(game)), original) {
		t.Error("the file wasn't restored exactly")
	}
}

func TestEnableModsRoundTripVariants(t *testing.T) {
	stock := fixture(t, "gameinfo.gi")

	variants := map[string][]byte{
		"LF":                  bytes.ReplaceAll(stock, []byte("\r\n"), []byte("\n")),
		"no trailing newline": bytes.TrimRight(stock, "\r\n"),
		"anchor last line":    []byte("SearchPaths\n{\n\tGame_LowViolence citadel_lv\n\tGame citadel\n}"),
		"brace on same line":  []byte("SearchPaths {\r\n  Game citadel\r\n  Game core\r\n}\r\n"),
		"quoted":              []byte("\"SearchPaths\"\n{\n\t\"Game\"\t\"citadel\"\n}\n"),
	}

	for name, data := range variants {
		t.Run(name, func(t *testing.T) {
			game := gameWith(t, data)

			if !enable(t, game, true) {
				t.Fatal("not changed")
			}

			if s := state(t, game); !s.Enabled || !s.Addons {
				t.Errorf("State = %+v", s)
			}

			after := readFile(t, GameInfoPath(game))

			if lines := bytes.Count(after, []byte("\n")); lines != bytes.Count(data, []byte("\n"))+4 {
				t.Errorf("%d lines, want %d", lines, bytes.Count(data, []byte("\n"))+4)
			}

			if bytes.Contains(data, []byte("\r\n")) != bytes.Contains(after, []byte("\r\n")) {
				t.Error("line endings changed")
			}

			disable(t, game)

			if got := readFile(t, GameInfoPath(game)); !bytes.Equal(got, data) {
				t.Errorf("not restored:\n%q\nwant\n%q", got, data)
			}
		})
	}
}

func TestEnableModsDMM(t *testing.T) {
	dmm := fixture(t, "gameinfo_dmm.gi")
	game := gameWith(t, dmm)

	if s := state(t, game); s != (State{Addons: true, DMM: true}) {
		t.Errorf("State = %+v", s)
	}

	enable(t, game, true)

	data := string(readFile(t, GameInfoPath(game)))

	// Ours goes above DMM's Game citadel, indented like it, and DMM's
	// addons line is left as the one and only.
	if !strings.Contains(data, "            Game                citadel/lockdeader\t// Lockdeader\r\n            Game                citadel\r\n") {
		t.Errorf("our line isn't in DMM's style above its Game citadel:\n%s", data)
	}

	if strings.Count(data, "citadel/addons") != 1 {
		t.Error("a second addons line was added")
	}

	// DMM's block has Mod and Write lines already.
	if strings.Count(data, "// Lockdeader") != 1 {
		t.Errorf("Mod or Write lines were added beside DMM's:\n%s", data)
	}

	if s := state(t, game); s != (State{Enabled: true, Addons: true, DMM: true}) {
		t.Errorf("State = %+v", s)
	}

	disable(t, game)

	if !bytes.Equal(readFile(t, GameInfoPath(game)), dmm) {
		t.Error("DMM's file wasn't restored exactly")
	}
}

func TestEnableModsUnparseable(t *testing.T) {
	files := map[string]string{
		"no SearchPaths":    "\"GameInfo\"\n{\n\tFileSystem\n\t{\n\t}\n}\n",
		"no Game citadel":   "SearchPaths\n{\n\tGame core\n}\n",
		"unbalanced braces": "SearchPaths\n{\n\tGame citadel\n",
		"commented out":     "SearchPaths\n{\n\t// Game citadel\n\tGame core\n}\n",
		"wrong path":        "SearchPaths\n{\n\tGame citadel_lv\n\tGame citadel/addons\n}\n",
	}

	for name, data := range files {
		t.Run(name, func(t *testing.T) {
			game := gameWith(t, []byte(data))

			if _, err := EnableMods(game, true, ""); !errors.Is(err, ErrNoSearchPaths) {
				t.Errorf("err = %v, want ErrNoSearchPaths", err)
			}

			if _, err := ReadGameInfo(game); !errors.Is(err, ErrNoSearchPaths) {
				t.Errorf("ReadGameInfo err = %v", err)
			}

			if got := readFile(t, GameInfoPath(game)); string(got) != data {
				t.Error("the file was changed")
			}
		})
	}
}

func TestEnableModsBackup(t *testing.T) {
	game := newGame(t)
	original := readFile(t, GameInfoPath(game))
	backup := filepath.Join(t.TempDir(), "backups", "gameinfo.gi")

	if _, err := EnableMods(game, false, backup); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(readFile(t, backup), original) {
		t.Error("the backup isn't the original")
	}

	// Only the first write makes a backup.
	os.WriteFile(backup, []byte("sentinel"), 0o644)

	if _, err := DisableMods(game, backup); err != nil {
		t.Fatal(err)
	}

	if _, err := EnableMods(game, true, backup); err != nil {
		t.Fatal(err)
	}

	if string(readFile(t, backup)) != "sentinel" {
		t.Error("the backup was overwritten")
	}
}

func TestEnableModsReadOnlyFile(t *testing.T) {
	game := newGame(t)
	path := GameInfoPath(game)

	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}

	if !enable(t, game, false) {
		t.Error("not changed")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm()&0o200 == 0 {
		t.Error("the file is still read-only")
	}
}

func TestGameInfoLeavesNoTemporaryFile(t *testing.T) {
	game := newGame(t)
	enable(t, game, true)
	disable(t, game)

	entries, err := os.ReadDir(filepath.Dir(GameInfoPath(game)))
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 || entries[0].Name() != "gameinfo.gi" {
		t.Errorf("citadel has %d entries", len(entries))
	}
}
