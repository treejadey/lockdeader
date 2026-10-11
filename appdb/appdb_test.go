package appdb

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func openTemp(t *testing.T) (*DB, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "Lockdeader.db")

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { d.Close() })

	return d, path
}

func TestOpenCreatesLatestSchema(t *testing.T) {
	d, path := openTemp(t)

	if v, err := d.Version(); err != nil || v != len(migrations) {
		t.Errorf("Version() = %d, %v; want %d", v, err, len(migrations))
	}

	d.Close()

	// Opening it again changes nothing.
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer again.Close()

	if v, _ := again.Version(); v != len(migrations) {
		t.Errorf("Version() after reopening = %d, want %d", v, len(migrations))
	}
}

func TestOpenWithSpaceInPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "with space", "Lockdeader.db")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	d.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database not created at %s: %v", path, err)
	}
}

func TestOpenNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Lockdeader.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}

	newer := len(migrations) + 1

	if _, err := raw.Exec("CREATE TABLE future (x INTEGER); PRAGMA user_version = " + strconv.Itoa(newer)); err != nil {
		t.Fatal(err)
	}

	raw.Close()

	if d, err := Open(path); !errors.Is(err, ErrNewerSchema) {
		if d != nil {
			d.Close()
		}

		t.Fatalf("Open() error = %v, want ErrNewerSchema", err)
	}

	// The newer database is left as it was.
	raw, _ = sql.Open("sqlite", path)
	defer raw.Close()

	var v int
	raw.QueryRow("PRAGMA user_version").Scan(&v)

	var tables int
	raw.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name = 'mods'").Scan(&tables)

	if v != newer || tables != 0 {
		t.Errorf("after Open: user_version = %d, mods tables = %d; want %d, 0", v, tables, newer)
	}
}

func TestMigrationsRunInOrder(t *testing.T) {
	saved := migrations
	t.Cleanup(func() { migrations = saved })

	path := filepath.Join(t.TempDir(), "Lockdeader.db")

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	d.Close()

	// A later release adds a migration: an existing database gets just that.
	var ran []int

	migrations = append(append([]migration(nil), saved...),
		func(ctx context.Context, tx *sql.Tx) error {
			ran = append(ran, len(saved)+1)

			_, err := tx.ExecContext(ctx, "ALTER TABLE mods ADD COLUMN notes TEXT NOT NULL DEFAULT ''")

			return err
		})

	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	if v, _ := d.Version(); v != len(saved)+1 || !reflect.DeepEqual(ran, []int{len(saved) + 1}) {
		t.Errorf("Version() = %d, ran %v; want %d, [%d]", v, ran, len(saved)+1, len(saved)+1)
	}
}

func TestFailedMigrationChangesNothing(t *testing.T) {
	saved := migrations
	t.Cleanup(func() { migrations = saved })

	path := filepath.Join(t.TempDir(), "Lockdeader.db")

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	d.Close()

	migrations = append(append([]migration(nil), saved...),
		execSQL("CREATE TABLE half (x INTEGER); THIS IS NOT SQL"))

	if d, err := Open(path); err == nil {
		d.Close()
		t.Fatal("Open() with a broken migration: want an error")
	}

	migrations = saved

	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	var tables int
	d.db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name = 'half'").Scan(&tables)

	if v, _ := d.Version(); v != len(saved) || tables != 0 {
		t.Errorf("after a failed migration: version %d, half-made tables %d; want %d, 0", v, tables, len(saved))
	}
}

func TestAudioState(t *testing.T) {
	ctx := context.Background()
	d, path := openTemp(t)

	if s, err := d.AudioState(ctx); err != nil || s != DefaultAudioState() {
		t.Errorf("AudioState() on a new database = %+v, %v; want defaults", s, err)
	}

	want := AudioState{Volume: 0.8, Muted: true}

	if err := d.SetAudioState(ctx, want); err != nil {
		t.Fatal(err)
	}

	if err := d.SetAudioState(ctx, AudioState{Volume: 1.5}); err == nil {
		t.Error("SetAudioState() with volume 1.5: want an error")
	}

	d.Close()

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	if s, err := d.AudioState(ctx); err != nil || s != want {
		t.Errorf("AudioState() after reopening = %+v, %v; want %+v", s, err, want)
	}

	// Importing doesn't overwrite what's there.
	if imported, err := d.ImportAudioState(ctx, AudioState{Volume: 0.1}); err != nil || imported {
		t.Errorf("ImportAudioState() over stored state = %v, %v; want false", imported, err)
	}

	if s, _ := d.AudioState(ctx); s != want {
		t.Errorf("AudioState() after importing = %+v, want %+v", s, want)
	}
}

func TestImportAudioStateIntoNewDatabase(t *testing.T) {
	ctx := context.Background()

	d, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	want := AudioState{Volume: 0.6}

	if imported, err := d.ImportAudioState(ctx, want); err != nil || !imported {
		t.Errorf("ImportAudioState() = %v, %v; want true", imported, err)
	}

	if s, _ := d.AudioState(ctx); s != want {
		t.Errorf("AudioState() = %+v, want %+v", s, want)
	}
}

func TestAudioStateOutOfRange(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	// e.g. edited by hand.
	if _, err := d.db.Exec("INSERT INTO app_state (key, value) VALUES ('audio.volume', 7.0)"); err != nil {
		t.Fatal(err)
	}

	if s, err := d.AudioState(ctx); err != nil || s.Volume != DefaultAudioState().Volume {
		t.Errorf("AudioState() = %+v, %v; want the default volume", s, err)
	}
}

func testMod() Mod {
	at := time.Unix(1_790_000_000, 0)

	return Mod{
		Section:      "Mod",
		GameBananaID: 654321,
		Name:         "Yamato Red",
		Author:       "someone",
		InstalledAt:  at,
		Thumbnail:    "https://images.gamebanana.com/img/ss/mods/530-90_yamato.jpg",
		UpdatedAt:    at.Add(-24 * time.Hour),
		NSFW:         true,
		Files: []ModFile{{
			GameBananaID: 1627082,
			Name:         "yamato_red_3f484.rar",
			Size:         24670571,
			MD5:          "5c87adef48470236adff65cbe693fbd6",
			UploadedAt:   at.Add(-time.Hour),
			DownloadedAt: at,
			VPKs: []VPK{
				{Path: "red/pak01_dir.vpk", SHA256: "aa11", Size: 1000, Enabled: true},
				{Path: "blue/pak01_dir.vpk", SHA256: "bb22", Size: 1200},
			},
		}},
	}
}

func TestMods(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	added, err := d.AddMod(ctx, testMod())
	if err != nil {
		t.Fatal(err)
	}

	if added.ID == 0 || added.Files[0].ID == 0 || added.Files[0].VPKs[1].ID == 0 {
		t.Fatalf("AddMod() didn't set the IDs: %+v", added)
	}

	mods, err := d.Mods(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(mods) != 1 || !reflect.DeepEqual(mods[0], added) {
		t.Fatalf("Mods() = %+v\nwant [%+v]", mods, added)
	}

	// The same mod, or the same file, can't be added twice.
	if _, err := d.AddMod(ctx, testMod()); err == nil {
		t.Error("adding the same mod again: want an error")
	}

	other := testMod()
	other.GameBananaID++

	if _, err := d.AddMod(ctx, other); err == nil {
		t.Error("adding another mod with the same file: want an error")
	}

	if mods, _ := d.Mods(ctx); len(mods) != 1 {
		t.Errorf("a failed AddMod left %d mods, want 1", len(mods))
	}

	if err := d.SetModEnabled(ctx, added.ID, true); err != nil {
		t.Fatal(err)
	}

	verified := time.Unix(1_790_000_500, 0)

	if err := d.SetVPKVerified(ctx, added.Files[0].VPKs[0].ID, verified); err != nil {
		t.Fatal(err)
	}

	mods, _ = d.Mods(ctx)
	if !mods[0].Enabled || !mods[0].Files[0].VPKs[0].VerifiedAt.Equal(verified) || !mods[0].Files[0].VPKs[1].VerifiedAt.IsZero() {
		t.Errorf("after enabling and verifying: %+v", mods[0])
	}

	if err := d.SetModEnabled(ctx, 9999, true); err == nil {
		t.Error("enabling a mod that isn't there: want an error")
	}

	// Removing a mod removes its files and VPKs too.
	if err := d.RemoveMod(ctx, added.ID); err != nil {
		t.Fatal(err)
	}

	var files, vpks int
	d.db.QueryRow("SELECT count(*) FROM mod_files").Scan(&files)
	d.db.QueryRow("SELECT count(*) FROM mod_vpks").Scan(&vpks)

	if mods, _ := d.Mods(ctx); len(mods) != 0 || files != 0 || vpks != 0 {
		t.Errorf("after RemoveMod: %d mods, %d files, %d VPKs; want none", len(mods), files, vpks)
	}
}

func TestAddModFiles(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	added, err := d.AddMod(ctx, testMod())
	if err != nil {
		t.Fatal(err)
	}

	more := []ModFile{{
		GameBananaID: 1627083,
		Name:         "yamato_blue.zip",
		Size:         500,
		MD5:          "00112233445566778899aabbccddeeff",
		UploadedAt:   time.Unix(1_790_000_100, 0),
		DownloadedAt: time.Unix(1_790_000_200, 0),
		VPKs:         []VPK{{Path: "pak01_dir.vpk", SHA256: "cc33", Size: 500, Enabled: true}},
	}}

	files, err := d.AddModFiles(ctx, added.ID, more)
	if err != nil {
		t.Fatal(err)
	}

	if files[0].ID == 0 || files[0].VPKs[0].ID == 0 {
		t.Fatalf("AddModFiles() didn't set the IDs: %+v", files)
	}

	mods, _ := d.Mods(ctx)
	want := append(added.Files, files...)
	if len(mods) != 1 || !reflect.DeepEqual(mods[0].Files, want) {
		t.Errorf("files after AddModFiles = %+v\nwant %+v", mods[0].Files, want)
	}

	// A file that's already stored can't be added again, and nothing of a
	// failed call is kept.
	again := append(append([]ModFile(nil), more...), testMod().Files...)
	again[0].GameBananaID = 1627084

	if _, err := d.AddModFiles(ctx, added.ID, again); err == nil {
		t.Error("adding a file that's already stored: want an error")
	}

	if mods, _ := d.Mods(ctx); len(mods[0].Files) != 2 {
		t.Errorf("a failed AddModFiles left %d files, want 2", len(mods[0].Files))
	}

	if _, err := d.AddModFiles(ctx, 9999, []ModFile{{GameBananaID: 1, Name: "x"}}); err == nil {
		t.Error("adding files to a mod that isn't there: want an error")
	}
}

func TestConfigureMod(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	added, err := d.AddMod(ctx, testMod())
	if err != nil {
		t.Fatal(err)
	}

	other := testMod()
	other.GameBananaID++
	other.Files[0].GameBananaID++

	otherAdded, err := d.AddMod(ctx, other)
	if err != nil {
		t.Fatal(err)
	}

	red, blue := added.Files[0].VPKs[0].ID, added.Files[0].VPKs[1].ID

	enabled := func() []bool {
		mods, _ := d.Mods(ctx)
		var on []bool

		for _, v := range mods[0].Files[0].VPKs {
			on = append(on, v.Enabled)
		}

		return on
	}

	if err := d.ConfigureMod(ctx, added.ID, nil, []int64{blue}); err != nil {
		t.Fatal(err)
	}

	if got := enabled(); !reflect.DeepEqual(got, []bool{false, true}) {
		t.Errorf("after picking blue: %v", got)
	}

	if err := d.ConfigureMod(ctx, added.ID, nil, []int64{red, blue}); err != nil {
		t.Fatal(err)
	}

	if got := enabled(); !reflect.DeepEqual(got, []bool{true, true}) {
		t.Errorf("after picking both: %v", got)
	}

	// None, or another mod's, is refused, and changes nothing.
	if err := d.ConfigureMod(ctx, added.ID, nil, nil); err == nil {
		t.Error("picking no VPKs: want an error")
	}

	if err := d.ConfigureMod(ctx, added.ID, nil, []int64{red, otherAdded.Files[0].VPKs[0].ID}); err == nil {
		t.Error("picking another mod's VPK: want an error")
	}

	// Removing another mod's file, or one that would leave nothing enabled,
	// is refused too.
	if err := d.ConfigureMod(ctx, added.ID, []int64{otherAdded.Files[0].ID}, []int64{red}); err == nil {
		t.Error("removing another mod's file: want an error")
	}

	if err := d.ConfigureMod(ctx, added.ID, []int64{added.Files[0].ID}, []int64{red}); err == nil {
		t.Error("enabling a VPK of a file being removed: want an error")
	}

	if got := enabled(); !reflect.DeepEqual(got, []bool{true, true}) {
		t.Errorf("after refused changes: %v", got)
	}

	// Removing a file removes its VPKs.
	more, err := d.AddModFiles(ctx, added.ID, []ModFile{{GameBananaID: 99, Name: "more.zip",
		VPKs: []VPK{{Path: "pak01_dir.vpk", SHA256: "dd44", Enabled: true}}}})
	if err != nil {
		t.Fatal(err)
	}

	if err := d.ConfigureMod(ctx, added.ID, []int64{added.Files[0].ID}, []int64{more[0].VPKs[0].ID}); err != nil {
		t.Fatal(err)
	}

	mods, _ := d.Mods(ctx)
	if len(mods[0].Files) != 1 || mods[0].Files[0].GameBananaID != 99 || !mods[0].Files[0].VPKs[0].Enabled {
		t.Errorf("after removing a file: %+v", mods[0].Files)
	}

	var left int
	d.db.QueryRow("SELECT count(*) FROM mod_vpks WHERE file_id = ?", added.Files[0].ID).Scan(&left)

	if left != 0 {
		t.Errorf("the removed file still has %d VPKs", left)
	}
}

func TestVPKInUse(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	added, err := d.AddMod(ctx, testMod())
	if err != nil {
		t.Fatal(err)
	}

	for sha, want := range map[string]bool{"aa11": true, "bb22": true, "cc33": false} {
		if got, err := d.VPKInUse(ctx, sha); err != nil || got != want {
			t.Errorf("VPKInUse(%q) = %v, %v; want %v", sha, got, err, want)
		}
	}

	d.RemoveMod(ctx, added.ID)

	if got, _ := d.VPKInUse(ctx, "aa11"); got {
		t.Error("VPKInUse() after removing the mod = true, want false")
	}
}

// Mods installed before version 2 have nothing to show for the columns it
// added, rather than failing to load.
func TestMigrationV2KeepsMods(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "Lockdeader.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := raw.Exec(schemaV1 + `
		PRAGMA user_version = 1;
		INSERT INTO mods (section, gamebanana_id, name, author, enabled, installed_at)
		VALUES ('Mod', 42, 'Old mod', 'someone', 1, 1790000000);`); err != nil {
		t.Fatal(err)
	}

	raw.Close()

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	mods, err := d.Mods(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(mods) != 1 || mods[0].Name != "Old mod" || !mods[0].Enabled ||
		mods[0].Thumbnail != "" || mods[0].UpdatedAt.Unix() != 0 || mods[0].NSFW {
		t.Errorf("Mods() after upgrading = %+v", mods)
	}
}

func TestModsLoadOrder(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	for i, order := range []int{2, 0, 1} {
		m := testMod()
		m.GameBananaID += i
		m.LoadOrder = order
		m.Files[0].GameBananaID += i

		if _, err := d.AddMod(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	mods, err := d.Mods(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var orders []int

	for _, m := range mods {
		orders = append(orders, m.LoadOrder)
	}

	if !reflect.DeepEqual(orders, []int{0, 1, 2}) {
		t.Errorf("load orders = %v, want [0 1 2]", orders)
	}
}

func TestStrictTypes(t *testing.T) {
	d, _ := openTemp(t)

	_, err := d.db.Exec(`INSERT INTO mods (section, gamebanana_id, name, installed_at) VALUES ('Mod', 'not a number', 'x', 0)`)
	if err == nil {
		t.Error("storing text in an INTEGER column: want an error")
	}
}

func TestUIState(t *testing.T) {
	ctx := context.Background()
	d, path := openTemp(t)

	if s, err := d.UIState(ctx); err != nil || s != DefaultUIState() {
		t.Errorf("UIState() on a new database = %+v, %v; want defaults", s, err)
	}

	want := UIState{Page: 1, ModsView: 1, InstalledView: 1, Section: "Sound", CategoryID: 47611,
		CategoryPath: "3,47611", ModsTopMod: 725782, ModsTopOffset: 40, OpenMod: 12,
		InstalledTop: 7, InstalledTopOffset: 3, SettingsY: 320}

	if err := d.SetUIState(ctx, want); err != nil {
		t.Fatal(err)
	}

	if err := d.SetUIState(ctx, UIState{}); err == nil {
		t.Error("SetUIState() with no section: want an error")
	}

	d.Close()

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer d.Close()

	if s, err := d.UIState(ctx); err != nil || s != want {
		t.Errorf("UIState() after reopening = %+v, %v; want %+v", s, err, want)
	}
}

func TestUIStateOutOfRange(t *testing.T) {
	ctx := context.Background()
	d, _ := openTemp(t)

	// e.g. edited by hand.
	if _, err := d.db.Exec("INSERT INTO app_state (key, value) VALUES ('ui.page', 9), ('ui.mods_view', -1), ('ui.settings_y', -5), ('ui.section', '')"); err != nil {
		t.Fatal(err)
	}

	if s, err := d.UIState(ctx); err != nil || s != DefaultUIState() {
		t.Errorf("UIState() = %+v, %v; want defaults", s, err)
	}
}

// Mods and files that aren't from GameBanana get ids of their own.
func TestAddLocalMod(t *testing.T) {
	d, _ := openTemp(t)
	ctx := context.Background()

	var ids []int

	for _, name := range []string{"one.vpk", "two.vpk"} {
		m, err := d.AddMod(ctx, Mod{Section: "Local", Name: name, Enabled: true, InstalledAt: time.Now(),
			Files: []ModFile{{Name: name, VPKs: []VPK{{Path: name, SHA256: name, Size: 1, Enabled: true}}}}})
		if err != nil {
			t.Fatal(err)
		}

		if m.GameBananaID != -int(m.ID) || m.Files[0].GameBananaID != -int(m.Files[0].ID) {
			t.Errorf("ids: mod %d/%d, file %d/%d", m.GameBananaID, m.ID, m.Files[0].GameBananaID, m.Files[0].ID)
		}

		ids = append(ids, m.GameBananaID)
	}

	if ids[0] == ids[1] {
		t.Errorf("two local mods share the id %d", ids[0])
	}

	mods, err := d.Mods(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(mods) != 2 || mods[0].GameBananaID != ids[0] || mods[1].Files[0].GameBananaID >= 0 {
		t.Errorf("Mods() = %+v", mods)
	}
}
