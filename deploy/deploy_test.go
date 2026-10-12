package deploy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/treejadey/lockdeader/vpk"
)

// newGame makes a fake Deadlock folder with the stock gameinfo.gi.
func newGame(t *testing.T) string {
	t.Helper()

	gameDir := filepath.Join(t.TempDir(), "Deadlock")
	citadel := filepath.Join(gameDir, "game", "citadel")

	if err := os.MkdirAll(citadel, 0o755); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile("testdata/gameinfo.gi")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(citadel, "gameinfo.gi"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	return gameDir
}

type store struct {
	t   *testing.T
	dir string
}

func newStore(t *testing.T) store {
	return store{t: t, dir: t.TempDir()}
}

// add writes a VPK holding files to the store, named by its SHA-256 as
// install.Store does.
func (s store) add(files map[string]string) File {
	s.t.Helper()

	var sources []vpk.Source

	for path, data := range files {
		sources = append(sources, vpk.Source{
			Path: path, CRC32: crc32.ChecksumIEEE([]byte(data)), Size: uint32(len(data)),
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(data)), nil },
		})
	}

	var b bytes.Buffer

	if err := vpk.Write(&b, sources); err != nil {
		s.t.Fatal(err)
	}

	sum := sha256.Sum256(b.Bytes())
	sha := hex.EncodeToString(sum[:])
	path := filepath.Join(s.dir, sha+".vpk")

	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		s.t.Fatal(err)
	}

	return File{SHA256: sha, Path: path, Size: int64(b.Len())}
}

// one makes a VPK with one file, unique to name.
func (s store) one(name string) File {
	return s.add(map[string]string{"materials/" + name + ".vmat_c": "data of " + name})
}

func folder(t *testing.T, gameDir string) map[string]int64 {
	t.Helper()

	entries, err := os.ReadDir(Dir(gameDir))
	if err != nil {
		t.Fatal(err)
	}

	sizes := make(map[string]int64, len(entries))

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}

		sizes[e.Name()] = info.Size()
	}

	return sizes
}

func apply(t *testing.T, gameDir string, files []File) Result {
	t.Helper()

	res, err := Apply(gameDir, files)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func sameContents(t *testing.T, a, b string) bool {
	t.Helper()

	return bytes.Equal(readFile(t, a), readFile(t, b))
}

func slotPath(gameDir string, i int) string {
	return filepath.Join(Dir(gameDir), slotName(i))
}

func TestApplyFromEmpty(t *testing.T) {
	game, s := newGame(t), newStore(t)
	files := []File{s.one("a"), s.one("b"), s.one("c")}

	res := apply(t, game, files)
	if res.Placed != 3 || res.Removed != 0 || res.Bundled != 0 {
		t.Errorf("Result = %+v", res)
	}

	want := map[string]int64{manifestName: folder(t, game)[manifestName]}

	for i, f := range files {
		want[slotName(i+1)] = f.Size

		if !sameContents(t, f.Path, slotPath(game, i+1)) {
			t.Errorf("slot %d doesn't hold %s", i+1, f.SHA256)
		}
	}

	if got := folder(t, game); !maps(got, want) {
		t.Errorf("folder = %v, want %v", got, want)
	}
}

func maps(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

func TestApplyIdempotent(t *testing.T) {
	game, s := newGame(t), newStore(t)
	files := []File{s.one("a"), s.one("b")}
	apply(t, game, files)

	manifestPath := filepath.Join(Dir(game), manifestName)
	before := readFile(t, manifestPath)
	info, _ := os.Stat(slotPath(game, 1))

	res := apply(t, game, files)
	if !nothing(res) {
		t.Errorf("second Apply did %+v", res)
	}

	if !bytes.Equal(readFile(t, manifestPath), before) {
		t.Error("the manifest was rewritten")
	}

	if after, _ := os.Stat(slotPath(game, 1)); !after.ModTime().Equal(info.ModTime()) {
		t.Error("a slot was rewritten")
	}
}

func TestApplyReorderReplaceRemove(t *testing.T) {
	game, s := newGame(t), newStore(t)
	a, b, c := s.one("a"), s.one("b"), s.one("c")
	apply(t, game, []File{a, b, c})

	// Swapping the first two rewrites only those slots.
	res := apply(t, game, []File{b, a, c})
	if res.Placed != 2 || res.Removed != 0 {
		t.Errorf("after a swap: %+v", res)
	}

	if !sameContents(t, b.Path, slotPath(game, 1)) || !sameContents(t, a.Path, slotPath(game, 2)) {
		t.Error("the slots weren't swapped")
	}

	// Dropping one renumbers densely.
	res = apply(t, game, []File{b, c})
	if res.Placed != 1 || res.Removed != 1 {
		t.Errorf("after a removal: %+v", res)
	}

	if !sameContents(t, c.Path, slotPath(game, 2)) {
		t.Error("slot 2 isn't c")
	}

	if _, err := os.Stat(slotPath(game, 3)); !errors.Is(err, os.ErrNotExist) {
		t.Error("slot 3 is still there")
	}

	// Nothing on leaves only the manifest.
	apply(t, game, nil)

	if got := folder(t, game); len(got) != 1 || got[manifestName] == 0 {
		t.Errorf("folder = %v", got)
	}
}

func TestApplyRemovesLeftovers(t *testing.T) {
	game, s := newGame(t), newStore(t)
	a := s.one("a")
	dir := Dir(game)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)

	for _, name := range []string{"pak05_dir.vpk.tmp", "stray.txt", "sub/x", "pak09_dir.vpk"} {
		os.WriteFile(filepath.Join(dir, name), []byte("old"), 0o644)
	}

	res := apply(t, game, []File{a})
	if res.Placed != 1 || res.Removed != 4 {
		t.Errorf("Result = %+v", res)
	}

	if got := folder(t, game); len(got) != 2 || got[slotName(1)] != a.Size {
		t.Errorf("folder = %v", got)
	}
}

func TestApplyRebuildsWhatTheManifestCantVouchFor(t *testing.T) {
	game, s := newGame(t), newStore(t)
	files := []File{s.one("a"), s.one("b")}
	apply(t, game, files)

	manifestPath := filepath.Join(Dir(game), manifestName)

	tests := map[string]struct {
		damage func(good []byte)
		placed int
	}{
		"missing manifest": {func([]byte) { os.Remove(manifestPath) }, 2},
		"corrupt manifest": {func([]byte) { os.WriteFile(manifestPath, []byte("{"), 0o644) }, 2},
		"manifest lying": {func(good []byte) {
			os.WriteFile(manifestPath, bytes.Replace(good, []byte(files[1].SHA256), []byte("0000"), 1), 0o644)
		}, 1},
		"wrong size on disk": {func([]byte) {
			// Removed first: the slot is a hard link to the store's file,
			// and writing into it would shorten that too.
			os.Remove(slotPath(game, 1))
			os.WriteFile(slotPath(game, 1), []byte("short"), 0o644)
		}, 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// The cases run in any order and share the folder, so each
			// starts from a sound one.
			apply(t, game, files)
			good := readFile(t, manifestPath)
			tc.damage(good)

			res := apply(t, game, files)
			if res.Placed != tc.placed {
				t.Errorf("placed %d slots, want %d", res.Placed, tc.placed)
			}

			for i, f := range files {
				if !sameContents(t, f.Path, slotPath(game, i+1)) {
					t.Errorf("slot %d is wrong", i+1)
				}
			}

			if !bytes.Equal(readFile(t, manifestPath), good) {
				t.Error("the manifest wasn't restored")
			}
		})
	}
}

func TestApplyTouchesNothingElse(t *testing.T) {
	game, s := newGame(t), newStore(t)
	citadel := filepath.Join(game, "game", "citadel")
	others := []string{filepath.Join(citadel, "addons", "pak01_dir.vpk"), filepath.Join(citadel, "pak01_dir.vpk"), filepath.Join(citadel, "gameinfo.gi")}
	os.MkdirAll(filepath.Join(citadel, "addons"), 0o755)
	os.WriteFile(others[0], []byte("someone else's mod"), 0o644)
	os.WriteFile(others[1], []byte("the game's own"), 0o644)
	before := map[string][]byte{}

	for _, p := range others {
		before[p] = readFile(t, p)
	}

	apply(t, game, []File{s.one("a")})
	apply(t, game, []File{s.one("b")})
	apply(t, game, nil)

	for _, p := range others {
		if !bytes.Equal(readFile(t, p), before[p]) {
			t.Errorf("%s changed", p)
		}
	}

	// Without game/citadel there's nothing to apply to.
	empty := t.TempDir()

	if _, err := Apply(empty, []File{s.one("c")}); err == nil {
		t.Error("applied to a folder that isn't the game's")
	}

	if _, err := os.Stat(filepath.Join(empty, "game")); !errors.Is(err, os.ErrNotExist) {
		t.Error("made a game folder")
	}
}

func TestApplyMissingFile(t *testing.T) {
	game, s := newGame(t), newStore(t)
	a, c := s.one("a"), s.one("c")
	b := File{SHA256: "gone", Path: filepath.Join(s.dir, "gone.vpk"), Size: 1}

	res, err := Apply(game, []File{a, b, c})
	if !errors.Is(err, ErrMissing) {
		t.Errorf("err = %v, want ErrMissing", err)
	}

	if !slices.Equal(res.Missing, []string{"gone"}) || res.Placed != 2 {
		t.Errorf("Result = %+v", res)
	}

	if !sameContents(t, c.Path, slotPath(game, 2)) {
		t.Error("the files after the missing one weren't numbered densely")
	}
}

func TestApplyBundles(t *testing.T) {
	game, s := newGame(t), newStore(t)
	var files []File

	for i := range 101 {
		files = append(files, s.one(fmt.Sprintf("m%03d", i)))
	}

	res := apply(t, game, files)
	if res.Placed != 99 || res.Bundled != 3 {
		t.Errorf("Result = %+v", res)
	}

	sizes := folder(t, game)
	if len(sizes) != 100 {
		t.Fatalf("folder has %d entries", len(sizes))
	}

	for i := range 98 {
		if !sameContents(t, files[i].Path, slotPath(game, i+1)) {
			t.Errorf("slot %d is wrong", i+1)
		}
	}

	// The last slot bundles the three lowest in the load order.
	f, err := os.Open(slotPath(game, 99))
	if err != nil {
		t.Fatal(err)
	}

	d, err := vpk.ReadDir(f)
	if err != nil {
		t.Fatal(err)
	}

	var paths []string

	for _, e := range d.Entries {
		data, _ := io.ReadAll(d.Open(e))
		paths = append(paths, e.Path)

		if want := "data of " + strings.TrimSuffix(strings.TrimPrefix(e.Path, "materials/"), ".vmat_c"); string(data) != want {
			t.Errorf("%s holds %q", e.Path, data)
		}

		if e.CRC32 != crc32.ChecksumIEEE(data) {
			t.Errorf("%s has the wrong CRC", e.Path)
		}
	}

	// Closed before the bundle is rebuilt below: Windows won't rename over
	// an open file (which is why nothing is applied while the game runs).
	f.Close()

	if want := []string{"materials/m098.vmat_c", "materials/m099.vmat_c", "materials/m100.vmat_c"}; !slices.Equal(paths, want) {
		t.Errorf("bundle holds %v, want %v", paths, want)
	}

	if res := apply(t, game, files); !nothing(res) {
		t.Errorf("second Apply did %+v", res)
	}

	// Dropping a member rebuilds only the bundle.
	res = apply(t, game, files[:100])
	if res.Placed != 1 || res.Bundled != 2 {
		t.Errorf("after a drop: %+v", res)
	}

	// Back under the limit, there's no bundle.
	res = apply(t, game, files[:50])
	if res.Placed != 0 || res.Removed != 49 {
		t.Errorf("after shrinking: %+v", res)
	}

	if !sameContents(t, files[49].Path, slotPath(game, 50)) {
		t.Error("slot 50 is wrong")
	}
}

func TestApplyLinksOrCopies(t *testing.T) {
	game, s := newGame(t), newStore(t)
	a := s.one("a")
	apply(t, game, []File{a})

	src, _ := os.Stat(a.Path)
	dst, _ := os.Stat(slotPath(game, 1))

	if !os.SameFile(src, dst) {
		t.Error("the slot isn't a hard link of the store file")
	}

	// Where linking doesn't work, e.g. across drives, the file is copied.
	link = func(string, string) error { return errors.New("cross-device") }
	t.Cleanup(func() { link = os.Link })

	b := s.one("b")
	apply(t, game, []File{a, b})

	src, _ = os.Stat(b.Path)
	dst, _ = os.Stat(slotPath(game, 2))

	if os.SameFile(src, dst) || !sameContents(t, b.Path, slotPath(game, 2)) {
		t.Error("the slot isn't a copy of the store file")
	}

	for name := range folder(t, game) {
		if strings.HasSuffix(name, tempSuffix) {
			t.Errorf("%s was left behind", name)
		}
	}
}

func TestApplyResumesAfterFailure(t *testing.T) {
	game, s := newGame(t), newStore(t)
	a, b, c := s.one("a"), s.one("b"), s.one("c")

	// A store file that can't be read fails its slot.
	broken := b
	broken.Path = t.TempDir()

	res, err := Apply(game, []File{a, broken, c})
	if err == nil {
		t.Fatal("no error")
	}

	if res.Placed != 1 {
		t.Errorf("placed %d, want 1", res.Placed)
	}

	res = apply(t, game, []File{a, b, c})
	if res.Placed != 2 {
		t.Errorf("the second Apply placed %d, want 2", res.Placed)
	}

	for i, f := range []File{a, b, c} {
		if !sameContents(t, f.Path, slotPath(game, i+1)) {
			t.Errorf("slot %d is wrong", i+1)
		}
	}
}

func nothing(res Result) bool {
	return res.Placed == 0 && res.Removed == 0 && res.Bundled == 0 && len(res.Missing) == 0
}
