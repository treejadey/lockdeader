package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/install"
)

// testArchive puts a small VPK for each body at pak01_dir.vpk if there's one,
// or in a folder per variant (0/, 1/, …).
func testArchive(t *testing.T, bodies ...string) []byte {
	t.Helper()

	var b bytes.Buffer
	z := zip.NewWriter(&b)

	for i, body := range bodies {
		var vpk bytes.Buffer
		binary.Write(&vpk, binary.LittleEndian, [7]uint32{0x55AA1234, 2})
		vpk.WriteString(body)

		name := "pak01_dir.vpk"

		if len(bodies) > 1 {
			name = fmt.Sprintf("%d/pak01_dir.vpk", i)
		}

		w, _ := z.Create(name)
		w.Write(vpk.Bytes())
	}

	z.Close()

	return b.Bytes()
}

type installTest struct {
	*installedMods
	db    *appdb.DB
	store install.Store
	srv   *httptest.Server
	// While it's false, the server's files are missing.
	serving atomic.Bool
	// While it's set, downloads hang until it's closed.
	hold atomic.Pointer[chan struct{}]
	// The files the server has, by path, and the files each mod has on
	// GameBanana, by id.
	mu       sync.Mutex
	files    map[string][]byte
	profiles map[int][]banana.File
}

func newInstallTest(t *testing.T) *installTest {
	t.Helper()

	db, err := appdb.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Close() })

	it := &installTest{db: db, store: install.Store{Dir: t.TempDir()},
		files: make(map[string][]byte), profiles: make(map[int][]banana.File)}

	it.serving.Store(true)

	it.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id int

		if _, err := fmt.Sscanf(r.URL.Path, "/Mod/%d/ProfilePage", &id); err == nil {
			it.mu.Lock()
			files := it.profiles[id]
			it.mu.Unlock()

			json.NewEncoder(w).Encode(map[string]any{"_idRow": id, "_aFiles": files})

			return
		}

		it.mu.Lock()
		data, ok := it.files[r.URL.Path]
		it.mu.Unlock()

		if !ok || !it.serving.Load() {
			http.NotFound(w, r)

			return
		}

		// Send the first byte, so the download has started, then wait.
		if hold := it.hold.Load(); hold != nil {
			w.Write(data[:1])
			w.(http.Flusher).Flush()

			select {
			case <-*hold:
			case <-r.Context().Done():
				return
			}

			data = data[1:]
		}

		w.Write(data)
	}))
	t.Cleanup(it.srv.Close)

	client := banana.NewClient()
	client.BaseURL = it.srv.URL
	client.HTTPClient = it.srv.Client()

	mainthread.Wait(func() { it.installedMods = newInstalledMods(client, db, it.store) })

	return it
}

// file serves an archive with a VPK for each body, and returns it as
// GameBanana lists it.
func (it *installTest) file(t *testing.T, id int, bodies ...string) banana.File {
	t.Helper()

	data := testArchive(t, bodies...)
	path := "/dl/" + string(rune('a'+id))
	it.mu.Lock()
	it.files[path] = data
	it.mu.Unlock()

	sum := md5.Sum(data)

	return banana.File{ID: id, Name: "mod.zip", Size: int64(len(data)), DownloadURL: it.srv.URL + path,
		MD5: hex.EncodeToString(sum[:]), DateAdded: 1_790_000_000}
}

func testInfo(id int) modInfo {
	return modInfo{section: banana.SectionMods, id: id, name: "Yamato Red", author: "someone",
		thumbnail: "https://images.gamebanana.com/a.jpg", updated: time.Unix(1_790_000_000, 0)}
}

// role is nil if there's no row for key.
func (it *installTest) role(key string, role int) *qt.QVariant {
	var v *qt.QVariant

	mainthread.Wait(func() {
		if row := it.rows[key]; row != nil {
			v = row.item.Data(role)
		}
	})

	return v
}

func (it *installTest) status(key string) string {
	v := it.role(key, roleInstalledStatus)
	if v == nil {
		return "(gone)"
	}

	return v.ToString()
}

func (it *installTest) message() string {
	return mainthread.Wait2(func() string { return it.Value("message").ToString() })
}

func (it *installTest) count(key string) int {
	return mainthread.Wait2(func() int { return it.Value(key).ToInt() })
}

func (it *installTest) waitForStatus(t *testing.T, key, want string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for it.status(key) != want {
		if time.Now().After(deadline) {
			t.Fatalf("status of %s = %q, want %q", key, it.status(key), want)
		}

		time.Sleep(5 * time.Millisecond)
	}
}

func (it *installTest) mods(t *testing.T) []appdb.Mod {
	t.Helper()

	mods, err := it.db.Mods(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return mods
}

func TestInstallMod(t *testing.T) {
	it := newInstallTest(t)
	red := it.file(t, 1, "red")
	key := installedKey(banana.SectionMods, 42)

	mainthread.Wait(func() { it.Install(testInfo(42), []banana.File{red}) })

	if got := it.role(key, roleInstalledName).ToString(); got != "Yamato Red" {
		t.Errorf("name while installing = %q", got)
	}

	if got := it.message(); got != "Downloading Yamato Red" {
		t.Errorf("message = %q", got)
	}

	it.waitForStatus(t, key, statusInstalled)

	mods := it.mods(t)
	if len(mods) != 1 || mods[0].GameBananaID != 42 || !mods[0].Enabled || mods[0].Thumbnail == "" ||
		len(mods[0].Files) != 1 || len(mods[0].Files[0].VPKs) != 1 {
		t.Fatalf("installed %+v", mods)
	}

	if _, err := os.Stat(it.store.Path(mods[0].Files[0].VPKs[0].SHA256)); err != nil {
		t.Errorf("the VPK isn't in the store: %v", err)
	}

	if !it.role(key, roleInstalledEnabled).ToBool() || it.count("total") != 1 || it.count("installing") != 0 {
		t.Errorf("after installing: enabled %v, total %d, installing %d",
			it.role(key, roleInstalledEnabled).ToBool(), it.count("total"), it.count("installing"))
	}

	// Installing the same file again does nothing; another file of the mod
	// is added to it.
	blue := it.file(t, 2, "blue")
	mainthread.Wait(func() { it.Install(testInfo(42), []banana.File{red}) })

	if got := it.status(key); got != statusInstalled {
		t.Errorf("status after installing the same file again = %q", got)
	}

	if got := it.message(); got != "Yamato Red is already installed" {
		t.Errorf("message after installing the same file again = %q", got)
	}

	mainthread.Wait(func() { it.Install(testInfo(42), []banana.File{red, blue}) })
	it.waitForStatus(t, key, statusInstalled)

	if mods := it.mods(t); len(mods) != 1 || len(mods[0].Files) != 2 {
		t.Errorf("after installing another file: %+v", mods)
	}

	mainthread.Wait(func() { it.SetEnabled(key, false) })

	if mods := it.mods(t); mods[0].Enabled || it.role(key, roleInstalledEnabled).ToBool() {
		t.Error("turning the mod off didn't")
	}
}

func TestInstallFailsAndRetries(t *testing.T) {
	it := newInstallTest(t)
	red := it.file(t, 1, "red")
	key := installedKey(banana.SectionMods, 7)

	it.serving.Store(false)
	mainthread.Wait(func() { it.Install(testInfo(7), []banana.File{red}) })
	it.waitForStatus(t, key, statusFailed)

	if it.role(key, roleInstalledError).ToString() == "" {
		t.Error("a failed install has no error")
	}

	if len(it.mods(t)) != 0 || it.count("installing") != 0 {
		t.Error("a failed install was recorded, or still counts as installing")
	}

	it.serving.Store(true)
	mainthread.Wait(func() { it.Retry(key) })
	it.waitForStatus(t, key, statusInstalled)

	if len(it.mods(t)) != 1 {
		t.Error("retrying didn't install the mod")
	}
}

func TestIdleAndFailedCallbacks(t *testing.T) {
	it := newInstallTest(t)

	var failed []string
	idle := 0

	mainthread.Wait(func() {
		it.onFailed = func(name string) { failed = append(failed, name) }
		it.onIdle = func() { idle++ }
	})

	missing := banana.File{ID: 9, Name: "x.vpk", DownloadURL: it.srv.URL + "/dl/missing"}
	info := testInfo(2)
	info.name = "Abrams Blue"

	mainthread.Wait(func() {
		it.Install(testInfo(1), []banana.File{it.file(t, 1, "one")})
		it.Install(info, []banana.File{missing})

		if it.Idle() {
			t.Error("Idle while mods are installing")
		}
	})

	it.waitForStatus(t, installedKey(banana.SectionMods, 1), statusInstalled)
	it.waitForStatus(t, installedKey(banana.SectionMods, 2), statusFailed)

	mainthread.Wait(func() {
		if !it.Idle() || idle != 1 {
			t.Errorf("once both are done, Idle() = %v and onIdle was called %d times, want once", it.Idle(), idle)
		}

		if len(failed) != 1 || failed[0] != "Abrams Blue" {
			t.Errorf("onFailed was called for %q, want the failed install only", failed)
		}
	})
}

func TestCancelInstall(t *testing.T) {
	it := newInstallTest(t)
	red := it.file(t, 1, "red")
	hold := make(chan struct{})
	it.hold.Store(&hold)
	defer close(hold)

	key := installedKey(banana.SectionMods, 9)
	mainthread.Wait(func() { it.Install(testInfo(9), []banana.File{red}) })
	it.waitForStatus(t, key, statusDownloading)

	if it.count("installing") != 1 {
		t.Errorf("installing = %d, want 1", it.count("installing"))
	}

	mainthread.Wait(func() { it.Cancel(key) })

	if got := it.status(key); got != "(gone)" {
		t.Errorf("status after cancelling = %q; want the row gone", got)
	}

	if it.count("total") != 0 || it.count("installing") != 0 {
		t.Errorf("after cancelling: total %d, installing %d", it.count("total"), it.count("installing"))
	}

	// The download stops, and nothing is recorded.
	deadline := time.Now().Add(5 * time.Second)

	for mainthread.Wait2(func() int { return it.running }) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the cancelled install is still running")
		}

		time.Sleep(5 * time.Millisecond)
	}

	if len(it.mods(t)) != 0 {
		t.Error("a cancelled install was recorded")
	}
}

func TestInstallQueue(t *testing.T) {
	it := newInstallTest(t)
	hold := make(chan struct{})
	it.hold.Store(&hold)

	var keys []string

	mainthread.Wait(func() {
		for id := 1; id <= maxRunningInstalls+1; id++ {
			it.Install(testInfo(id), []banana.File{it.file(t, id, "mod")})
			keys = append(keys, installedKey(banana.SectionMods, id))
		}
	})

	last := keys[len(keys)-1]
	it.waitForStatus(t, keys[0], statusDownloading)

	if got := it.status(last); got != statusQueued {
		t.Errorf("status of the install past the limit = %q, want queued", got)
	}

	if got := it.message(); got != "Waiting to download Yamato Red" {
		t.Errorf("message for the install past the limit = %q", got)
	}

	it.hold.Store(nil)
	close(hold)

	for _, key := range keys {
		it.waitForStatus(t, key, statusInstalled)
	}

	if mods := it.mods(t); len(mods) != len(keys) || mods[len(mods)-1].LoadOrder != len(keys)-1 {
		t.Errorf("installed %+v; want each after the last", mods)
	}
}

func TestUninstallAndSearch(t *testing.T) {
	it := newInstallTest(t)
	shared := it.file(t, 1, "shared")
	red := it.file(t, 2, "shared") // the same VPK as the first

	mainthread.Wait(func() {
		it.Install(testInfo(1), []banana.File{shared})

		info := testInfo(2)
		info.name = "Abrams Blue"
		it.Install(info, []banana.File{red})
	})

	first, second := installedKey(banana.SectionMods, 1), installedKey(banana.SectionMods, 2)
	it.waitForStatus(t, first, statusInstalled)
	it.waitForStatus(t, second, statusInstalled)

	filtered := func() int {
		return mainthread.Wait2(func() int { return it.Filtered.RowCount(qt.NewQModelIndex()) })
	}

	mainthread.Wait(func() { it.Filtered.SetFilterFixedString("abrams") })

	if n := filtered(); n != 1 {
		t.Errorf("searching for one mod lists %d", n)
	}

	mainthread.Wait(func() { it.Filtered.SetFilterFixedString("") })

	sha := it.mods(t)[0].Files[0].VPKs[0].SHA256
	mainthread.Wait(func() { it.Uninstall(first) })

	// It leaves the list at once, but stays installed while it can be undone.
	if it.status(first) != "(gone)" || filtered() != 1 || it.count("total") != 1 || len(it.mods(t)) != 2 {
		t.Fatal("uninstalling didn't take the mod off the list, or removed it before Undo was too late")
	}

	mainthread.Wait(func() { it.FinishUninstall() })

	if len(it.mods(t)) != 1 {
		t.Fatal("uninstalling didn't remove the mod")
	}

	// The other mod still has the VPK.
	if _, err := os.Stat(it.store.Path(sha)); err != nil {
		t.Errorf("a VPK another mod has was deleted: %v", err)
	}

	// Its message going makes Undo too late.
	mainthread.Wait(func() {
		it.Uninstall(second)
		it.messageGone(it.Value("messageId").ToInt())
	})

	if _, err := os.Stat(it.store.Path(sha)); !os.IsNotExist(err) {
		t.Errorf("the VPK is still there after uninstalling every mod that had it: %v", err)
	}
}

func TestUndoUninstall(t *testing.T) {
	it := newInstallTest(t)

	mainthread.Wait(func() {
		it.Install(testInfo(1), []banana.File{it.file(t, 1, "one")})

		info := testInfo(2)
		info.name = "Abrams Blue"
		it.Install(info, []banana.File{it.file(t, 2, "two")})
	})

	first, second := installedKey(banana.SectionMods, 1), installedKey(banana.SectionMods, 2)
	it.waitForStatus(t, first, statusInstalled)
	it.waitForStatus(t, second, statusInstalled)

	row := func(key string) int {
		return mainthread.Wait2(func() int { return it.rows[key].item.Row() })
	}

	before := row(first)
	var messageID int

	mainthread.Wait(func() {
		it.Uninstall(first)
		messageID = it.Value("messageId").ToInt()
	})

	if got := it.message(); got != "Yamato Red was uninstalled" {
		t.Errorf("message = %q", got)
	}

	if !mainthread.Wait2(func() bool { return it.Value("messageUndoable").ToBool() }) {
		t.Error("the uninstall message doesn't offer Undo")
	}

	// Undo for a message that isn't the uninstall's does nothing.
	mainthread.Wait(func() { it.Undo(messageID - 1) })

	if it.status(first) != "(gone)" {
		t.Fatal("Undo for another message put the mod back")
	}

	mainthread.Wait(func() { it.Undo(messageID) })

	if it.status(first) != statusInstalled || row(first) != before || it.count("total") != 2 {
		t.Fatalf("Undo didn't put the mod back where it was: status %q, row %d (was %d)", it.status(first), row(first), before)
	}

	// It's no longer pending, so its message going doesn't remove it.
	mainthread.Wait(func() {
		it.messageGone(messageID)
		it.FinishUninstall()
	})

	if len(it.mods(t)) != 2 {
		t.Error("a mod put back by Undo was removed anyway")
	}
}

func TestRemoveUnusedVPKs(t *testing.T) {
	it := newInstallTest(t)
	mainthread.Wait(func() { it.Install(testInfo(1), []banana.File{it.file(t, 1, "kept")}) })
	it.waitForStatus(t, installedKey(banana.SectionMods, 1), statusInstalled)

	// Left behind by an install that never finished.
	var vpk bytes.Buffer
	binary.Write(&vpk, binary.LittleEndian, [7]uint32{0x55AA1234, 2})

	orphan, _, err := it.store.Add(&vpk)
	if err != nil {
		t.Fatal(err)
	}

	if err := removeUnusedVPKs(context.Background(), it.db, it.store); err != nil {
		t.Fatal(err)
	}

	shas, _ := it.store.VPKs()
	if len(shas) != 1 || shas[0] == orphan {
		t.Errorf("store has %v after removing unused VPKs; want just the installed one", shas)
	}
}

type configOption struct {
	id        string
	detail    string
	picked    bool
	installed bool
}

func (it *installTest) configuration(t *testing.T, key string) []configOption {
	t.Helper()

	var list []configOption

	mainthread.Wait(func() {
		it.configure(key)

		config := it.Value("configuration").ToMap()

		if got := config["key"]; got.ToString() != key {
			t.Errorf("configuration is for %q, want %q", got.ToString(), key)
		}

		opts := config["options"]

		for _, v := range opts.ToList() {
			m := v.ToMap()
			id, detail, picked, installed := m["id"], m["detail"], m["picked"], m["installed"]
			list = append(list, configOption{id.ToString(), detail.ToString(), picked.ToBool(), installed.ToBool()})
		}
	})

	return list
}

func TestConfigure(t *testing.T) {
	it := newInstallTest(t)
	key := installedKey(banana.SectionMods, 3)
	variants := it.file(t, 1, "red", "blue")
	flag := it.file(t, 2, "flag")
	flag.Description = "Flag only"
	music := it.file(t, 3, "music")
	info := testInfo(3)
	info.available = []banana.File{variants, flag, music}

	mainthread.Wait(func() { it.Install(info, []banana.File{variants, flag}) })
	it.waitForStatus(t, key, statusInstalled)

	if n := it.role(key, roleInstalledVariants).ToInt(); n != 4 {
		t.Errorf("variants = %d, want 4", n)
	}

	opts := it.configuration(t, key)
	if len(opts) != 4 || opts[0].detail != "0/pak01_dir.vpk" || opts[1].detail != "1/pak01_dir.vpk" ||
		opts[2].detail != "Flag only" || opts[3].id != "file:3" ||
		!opts[0].picked || !opts[1].picked || !opts[2].picked || opts[3].picked || opts[3].installed {
		t.Fatalf("options = %+v", opts)
	}

	red, blue := opts[0].id, opts[1].id
	flagSHA := it.mods(t)[0].Files[1].VPKs[0].SHA256

	// Picking music downloads it; only then is the flag removed.
	// The status is read in the same call, or the download could finish
	// before it is.
	var status string

	mainthread.Wait(func() {
		it.Configure(key, []string{blue, "file:3"})
		status = it.status(key)
	})

	if status == statusInstalled {
		t.Error("configuring with a file to download didn't start an install")
	}

	it.waitForStatus(t, key, statusInstalled)
	files := it.mods(t)[0].Files
	if len(files) != 2 || files[0].GameBananaID != 1 || files[1].GameBananaID != 3 ||
		files[0].VPKs[0].Enabled || !files[0].VPKs[1].Enabled || !files[1].VPKs[0].Enabled {
		t.Errorf("files after picking blue and music: %+v", files)
	}

	if _, err := os.Stat(it.store.Path(flagSHA)); !os.IsNotExist(err) {
		t.Errorf("the removed flag's VPK is still in the store: %v", err)
	}

	// Changes to installed options happen straight away.
	mainthread.Wait(func() { it.Configure(key, []string{red}) })
	files = it.mods(t)[0].Files
	if len(files) != 1 || !files[0].VPKs[0].Enabled || files[0].VPKs[1].Enabled {
		t.Errorf("files after picking red: %+v", files)
	}

	// Picking none, or only options that aren't the mod's, is refused.
	mainthread.Wait(func() {
		it.Configure(key, nil)
		it.Configure(key, []string{"vpk:9999"})
	})

	if files := it.mods(t)[0].Files; len(files) != 1 || !files[0].VPKs[0].Enabled {
		t.Errorf("files after refused changes: %+v", files)
	}
}

func TestLookUpFiles(t *testing.T) {
	it := newInstallTest(t)
	key := installedKey(banana.SectionMods, 5)
	only := it.file(t, 1, "only")
	other := it.file(t, 2, "other")
	it.mu.Lock()
	it.profiles[5] = []banana.File{only, other}
	it.mu.Unlock()

	// Installed without knowing its other files: they're looked up.
	mainthread.Wait(func() { it.Install(testInfo(5), []banana.File{only}) })
	it.waitForStatus(t, key, statusInstalled)

	deadline := time.Now().Add(5 * time.Second)

	for it.role(key, roleInstalledVariants).ToInt() != 2 {
		if time.Now().After(deadline) {
			t.Fatalf("variants = %d, want 2 once the files are looked up", it.role(key, roleInstalledVariants).ToInt())
		}

		time.Sleep(5 * time.Millisecond)
	}
}
