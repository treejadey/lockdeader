package deploy

import (
	"bytes"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/treejadey/lockdeader/vpk"
)

// fakeBase writes a pak01_dir.vpk with files into a fake game folder.
func fakeBase(t *testing.T, game string, files map[string]string) {
	t.Helper()

	var sources []vpk.Source

	for p, data := range files {
		sources = append(sources, vpk.Source{
			Path: p, CRC32: crc32.ChecksumIEEE([]byte(data)), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(data)), nil },
		})
	}

	var b bytes.Buffer

	if err := vpk.Write(&b, sources); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(BasePak(game)), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(BasePak(game), b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBase(t *testing.T) {
	game := t.TempDir()
	fakeBase(t, game, map[string]string{"soundevents/Player.vsndevts_c": "the game's"})

	b, err := ReadBase(game)
	if err != nil {
		t.Fatal(err)
	}

	data, ok, err := b.File("soundevents/player.vsndevts_c")
	if err != nil || !ok || string(data) != "the game's" {
		t.Errorf("File = %q, %v, %v; want the game's copy", data, ok, err)
	}

	if _, ok, err := b.File("soundevents/none.vsndevts_c"); ok || err != nil {
		t.Errorf("File of a file the game hasn't = %v, %v", ok, err)
	}

	if !b.Current(game) || b.Current(t.TempDir()) {
		t.Error("Current is wrong about which game it is")
	}

	// A game update replaces pak01_dir.vpk.
	stamp := b.Stamp()
	fakeBase(t, game, map[string]string{"soundevents/player.vsndevts_c": "updated, and longer"})

	later := time.Now().Add(time.Minute)
	os.Chtimes(BasePak(game), later, later)

	if b.Current(game) {
		t.Error("Current after the game updated")
	}

	if b2, err := ReadBase(game); err != nil || b2.Stamp() == stamp {
		t.Errorf("Stamp after the game updated = %v, %v; want a new one", b2, err)
	}

	if _, err := ReadBase(t.TempDir()); err == nil {
		t.Error("ReadBase found a game in an empty folder")
	}
}
