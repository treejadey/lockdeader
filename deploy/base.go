package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/treejadey/lockdeader/vpk"
)

// A Base reads files of the base game, from game/citadel/pak01_dir.vpk and
// its chunks, which mods' tables are merged over. Reading the directory
// takes about a tenth of a second, so it's kept until the game updates.
type Base struct {
	path    string
	modTime time.Time
	size    int64
	dir     *vpk.Dir
	files   map[string]vpk.Entry
}

// BasePak is the base game's directory VPK.
func BasePak(gameDir string) string {
	return filepath.Join(gameDir, "game", "citadel", "pak01_dir.vpk")
}

// ReadBase reads the directory of the base game in gameDir.
func ReadBase(gameDir string) (*Base, error) {
	path := BasePak(gameDir)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("deploy: %w", err)
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("deploy: %w", err)
	}

	dir, err := vpk.ReadDir(f)
	if err != nil {
		return nil, fmt.Errorf("deploy: %s: %w", path, err)
	}

	b := &Base{path: path, modTime: info.ModTime(), size: info.Size(), dir: dir,
		files: make(map[string]vpk.Entry, len(dir.Entries))}

	for _, e := range dir.Entries {
		b.files[strings.ToLower(e.Path)] = e
	}

	return b, nil
}

// Current says whether b is still the game in gameDir: the same folder, and
// a pak01_dir.vpk that no update has replaced since it was read.
func (b *Base) Current(gameDir string) bool {
	if BasePak(gameDir) != b.path {
		return false
	}

	info, err := os.Stat(b.path)

	return err == nil && info.ModTime().Equal(b.modTime) && info.Size() == b.size
}

// File returns a file of the base game, by its path in lower case, or false
// if the game has none.
func (b *Base) File(path string) ([]byte, bool, error) {
	e, ok := b.files[path]
	if !ok {
		return nil, false, nil
	}

	data, err := b.dir.ReadFile(e, b.path)
	if err != nil {
		return nil, true, fmt.Errorf("deploy: %w", err)
	}

	return data, true, nil
}

// Stamp tells versions of the base game apart, e.g. to key a cache.
func (b *Base) Stamp() string {
	return fmt.Sprintf("%s|%d|%d", b.path, b.modTime.UnixNano(), b.size)
}
