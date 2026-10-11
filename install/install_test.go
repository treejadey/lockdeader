package install

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/treejadey/lockdeader/banana"
)

// vpkBytes returns a small file that starts like a VPK, followed by body.
func vpkBytes(body string) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [7]uint32{vpkMagic, 2})
	b.WriteString(body)
	return b.Bytes()
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func newStore(t *testing.T) Store {
	t.Helper()
	return Store{Dir: t.TempDir()}
}

// storeFiles lists the names of the files in the store.
func storeFiles(t *testing.T, s Store) []string {
	t.Helper()
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestStoreAdd(t *testing.T) {
	s := newStore(t)
	data := vpkBytes("hello")

	got, size, err := s.Add(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got != sha(data) || size != int64(len(data)) {
		t.Errorf("Add() = %s, %d; want %s, %d", got, size, sha(data), len(data))
	}
	if stored, _ := os.ReadFile(s.Path(got)); !bytes.Equal(stored, data) {
		t.Errorf("stored %q, want %q", stored, data)
	}

	// Adding it again keeps the one copy.
	if again, _, err := s.Add(bytes.NewReader(data)); err != nil || again != got {
		t.Errorf("adding again = %s, %v", again, err)
	}
	if names := storeFiles(t, s); !reflect.DeepEqual(names, []string{got + ".vpk"}) {
		t.Errorf("store has %v, want just the VPK", names)
	}

	for _, notVPK := range []string{"", "abc", "PK\x03\x04 a zip, not a VPK"} {
		if _, _, err := s.Add(bytes.NewReader([]byte(notVPK))); !errors.Is(err, ErrNotVPK) {
			t.Errorf("Add(%q) error = %v, want ErrNotVPK", notVPK, err)
		}
	}
	if names := storeFiles(t, s); len(names) != 1 {
		t.Errorf("failed adds left files behind: %v", names)
	}
}

func TestStoreCleanUpAndRemove(t *testing.T) {
	s := newStore(t)
	sum, _, err := s.Add(bytes.NewReader(vpkBytes("keep")))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"123.partial", "4567.tmp"} {
		os.WriteFile(filepath.Join(s.Dir, name), []byte("left over"), 0o644)
	}
	if err := s.CleanUp(); err != nil {
		t.Fatal(err)
	}
	if names := storeFiles(t, s); !reflect.DeepEqual(names, []string{sum + ".vpk"}) {
		t.Errorf("after CleanUp, store has %v; want just the VPK", names)
	}
	if shas, _ := s.VPKs(); !reflect.DeepEqual(shas, []string{sum}) {
		t.Errorf("VPKs() = %v, want [%s]", shas, sum)
	}

	if err := s.Remove(sum); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(sum); err != nil {
		t.Errorf("removing a VPK that isn't there: %v", err)
	}
	if names := storeFiles(t, s); len(names) != 0 {
		t.Errorf("after Remove, store has %v", names)
	}
}

// zipFile writes a zip with the given files, in order, and returns its path.
func zipFile(t *testing.T, files ...[2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mod.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, file := range files {
		w, _ := z.Create(file[0])
		w.Write([]byte(file[1]))
	}
	z.Close()
	f.Close()
	return path
}

func TestExtractZip(t *testing.T) {
	s := newStore(t)
	red, blue := vpkBytes("red"), vpkBytes("blue")
	archive := zipFile(t,
		[2]string{"readme.txt", "hi"},
		[2]string{"Red/pak01_dir.vpk", string(red)},
		[2]string{"__MACOSX/Red/._pak01_dir.vpk", "resource fork"},
		[2]string{"Blue\\PAK01_DIR.VPK", string(blue)},
	)

	vpks, err := Extract(archive, "mod.zip", s)
	if err != nil {
		t.Fatal(err)
	}
	want := []VPK{
		{Path: "Red/pak01_dir.vpk", SHA256: sha(red), Size: int64(len(red))},
		{Path: "Blue/PAK01_DIR.VPK", SHA256: sha(blue), Size: int64(len(blue))},
	}
	if !reflect.DeepEqual(vpks, want) {
		t.Errorf("Extract() = %+v\nwant %+v", vpks, want)
	}
}

func TestExtract7z(t *testing.T) {
	s := newStore(t)
	vpks, err := Extract("testdata/mod.7z", "mod.7z", s)
	if err != nil {
		t.Fatal(err)
	}
	data := vpkBytes("7z variant")
	want := []VPK{{Path: "skins/pak01_dir.vpk", SHA256: sha(data), Size: int64(len(data))}}
	if !reflect.DeepEqual(vpks, want) {
		t.Errorf("Extract() = %+v\nwant %+v", vpks, want)
	}
}

// rarFile writes a RAR 5 archive with the given files stored uncompressed,
// as `rar a -m0` would, and returns its path.
func rarFile(t *testing.T, files ...[2]string) string {
	t.Helper()
	vint := func(b []byte, v uint64) []byte {
		for v >= 0x80 {
			b = append(b, byte(v)|0x80)
			v >>= 7
		}
		return append(b, byte(v))
	}
	var out bytes.Buffer
	// Each block is its header's CRC32, its size, then the header.
	block := func(header []byte) {
		sized := vint(nil, uint64(len(header)))
		sized = append(sized, header...)
		binary.Write(&out, binary.LittleEndian, crc32.ChecksumIEEE(sized))
		out.Write(sized)
	}

	out.WriteString("Rar!\x1a\x07\x01\x00")
	block([]byte{1, 0, 0}) // main archive header: no flags
	for _, file := range files {
		data := []byte(file[1])
		h := vint(nil, 2)              // file header
		h = vint(h, 2)                 // it has a data area
		h = vint(h, uint64(len(data))) // of this size
		h = vint(h, 4)                 // file flags: has a CRC32
		h = vint(h, uint64(len(data))) // unpacked size
		h = vint(h, 0o644)             // attributes
		h = binary.LittleEndian.AppendUint32(h, crc32.ChecksumIEEE(data))
		h = vint(h, 0) // compression: stored
		h = vint(h, 1) // made on Unix
		h = vint(h, uint64(len(file[0])))
		h = append(h, file[0]...)
		block(h)
		out.Write(data)
	}
	block([]byte{5, 0, 0}) // end of archive

	path := filepath.Join(t.TempDir(), "mod.rar")
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractRar(t *testing.T) {
	s := newStore(t)
	data := vpkBytes("rar")
	archive := rarFile(t, [2]string{"notes.txt", "hi"}, [2]string{"mod/pak01_dir.vpk", string(data)})

	vpks, err := Extract(archive, "mod.rar", s)
	if err != nil {
		t.Fatal(err)
	}
	want := []VPK{{Path: "mod/pak01_dir.vpk", SHA256: sha(data), Size: int64(len(data))}}
	if !reflect.DeepEqual(vpks, want) {
		t.Errorf("Extract() = %+v\nwant %+v", vpks, want)
	}
}

func TestExtractBareVPK(t *testing.T) {
	s := newStore(t)
	data := vpkBytes("bare")
	path := filepath.Join(t.TempDir(), "123.partial")
	os.WriteFile(path, data, 0o644)

	vpks, err := Extract(path, "pak01_dir.vpk", s)
	if err != nil {
		t.Fatal(err)
	}
	want := []VPK{{Path: "pak01_dir.vpk", SHA256: sha(data), Size: int64(len(data))}}
	if !reflect.DeepEqual(vpks, want) {
		t.Errorf("Extract() = %+v\nwant %+v", vpks, want)
	}
}

func TestExtractErrors(t *testing.T) {
	notArchive := filepath.Join(t.TempDir(), "mod.exe")
	os.WriteFile(notArchive, []byte("MZ not an archive"), 0o644)

	tests := []struct {
		name    string
		archive string
		want    error
	}{
		{"no VPKs", zipFile(t, [2]string{"readme.txt", "hi"}), ErrNoVPKs},
		{"chunked", zipFile(t,
			[2]string{"pak01_dir.vpk", string(vpkBytes("dir"))},
			[2]string{"pak01_000.vpk", "chunk"}), ErrChunkedVPK},
		{"not a VPK inside", zipFile(t, [2]string{"pak01_dir.vpk", "text"}), ErrNotVPK},
		{"unknown format", notArchive, ErrUnknownFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Extract(tt.archive, "mod", newStore(t)); !errors.Is(err, tt.want) {
				t.Errorf("Extract() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSpeedMeterPace(t *testing.T) {
	const mbps = 1e6 / 8 // bytes per second
	start := time.Unix(1_790_000_000, 0)

	// feed reports a steady speed for a few seconds, and returns the pace
	// after each second.
	feed := func(m *SpeedMeter, at *time.Time, received *int64, speed float64, seconds int) []Pace {
		var paces []Pace
		for i := 0; i < seconds*10; i++ {
			*at = at.Add(100 * time.Millisecond)
			*received += int64(speed / 10)
			m.Add(*at, *received)
			if i%10 == 9 {
				paces = append(paces, m.Pace())
			}
		}
		return paces
	}

	var m SpeedMeter
	if m.Pace() != PaceNormal {
		t.Errorf("Pace() before any data = %v, want normal", m.Pace())
	}
	at, received := start, int64(0)

	// Too soon to tell.
	m.Add(at, 0)
	at = at.Add(100 * time.Millisecond)
	m.Add(at, int64(mbps)) // 80 Mbps for a moment
	if m.Pace() != PaceNormal {
		t.Errorf("Pace() after 100 ms = %v, want normal", m.Pace())
	}

	steps := []struct {
		speed float64 // Mbps
		want  Pace    // after a few seconds
	}{
		{5, PaceSlow},
		{18, PaceSlow},   // not clearly past the boundary
		{25, PaceNormal}, // clearly past it
		{100, PaceFast},
		{57, PaceFast}, // not clearly back under
		{50, PaceNormal},
		{40, PaceNormal},
		{16, PaceSlow},
	}
	for _, step := range steps {
		paces := feed(&m, &at, &received, step.speed*mbps, 4)
		if got := paces[len(paces)-1]; got != step.want {
			t.Errorf("at %v Mbps: Pace() = %v, want %v (each second: %v)", step.speed, got, step.want, paces)
		}
		if bps := m.BytesPerSecond(); bps < 0.95*step.speed*mbps || bps > 1.05*step.speed*mbps {
			t.Errorf("at %v Mbps: BytesPerSecond() = %.0f", step.speed, bps)
		}
	}
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func TestFetch(t *testing.T) {
	red, blue := vpkBytes("red"), vpkBytes("blue")
	archiveDir := t.TempDir()
	first, _ := os.ReadFile(zipFile(t, [2]string{"red/pak01_dir.vpk", string(red)}))
	os.WriteFile(filepath.Join(archiveDir, "first.zip"), first, 0o644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dl/1":
			w.Write(first)
		case "/dl/2":
			w.Write(blue)
		case "/dl/3":
			w.Write(blue[:4]) // cut short
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := banana.NewClient()
	client.HTTPClient = srv.Client()

	files := []banana.File{
		{ID: 1, Name: "first.zip", Size: int64(len(first)), DownloadURL: srv.URL + "/dl/1", MD5: md5Hex(first), DateAdded: 1_790_000_000},
		{ID: 2, Name: "pak02_dir.vpk", Size: int64(len(blue)), DownloadURL: srv.URL + "/dl/2", MD5: md5Hex(blue)},
	}

	s := newStore(t)
	var statuses []Status
	got, err := Fetch(context.Background(), client, s, files, func(st Status) { statuses = append(statuses, st) })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].GameBananaID != 1 || got[0].MD5 != md5Hex(first) || got[0].UploadedAt.Unix() != 1_790_000_000 ||
		len(got[0].VPKs) != 1 || got[0].VPKs[0].SHA256 != sha(red) || !got[0].VPKs[0].Enabled ||
		len(got[1].VPKs) != 1 || got[1].VPKs[0].Path != "pak02_dir.vpk" || got[1].VPKs[0].SHA256 != sha(blue) {
		t.Errorf("Fetch() = %+v", got)
	}
	// The archives are gone; only the VPKs are left.
	names := storeFiles(t, s)
	slices.Sort(names)
	want := []string{sha(red) + ".vpk", sha(blue) + ".vpk"}
	slices.Sort(want)
	if !reflect.DeepEqual(names, want) {
		t.Errorf("store has %v, want %v", names, want)
	}

	total := int64(len(first) + len(blue))
	if len(statuses) < 2 || statuses[0] != (Status{Stage: Downloading, Total: total}) ||
		statuses[len(statuses)-1].Stage != Extracting || statuses[len(statuses)-1].Received != total {
		t.Errorf("reported %+v", statuses)
	}

	// A download that doesn't match its checksum.
	damaged := []banana.File{{ID: 3, Name: "x.vpk", Size: int64(len(blue)), DownloadURL: srv.URL + "/dl/3", MD5: md5Hex(blue)}}
	if _, err := Fetch(context.Background(), client, s, damaged, func(Status) {}); !errors.Is(err, ErrChecksum) {
		t.Errorf("damaged download: error = %v, want ErrChecksum", err)
	}
	missing := []banana.File{{ID: 4, Name: "x.vpk", DownloadURL: srv.URL + "/dl/4"}}
	if _, err := Fetch(context.Background(), client, s, missing, func(Status) {}); err == nil {
		t.Error("missing download: want an error")
	}
	if names := storeFiles(t, s); len(names) != 2 {
		t.Errorf("failed fetches left files behind: %v", names)
	}
}

func TestFetchCancelAndStall(t *testing.T) {
	// Sends a little, then nothing until the client gives up.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("start"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	client := banana.NewClient()
	client.HTTPClient = srv.Client()
	files := []banana.File{{ID: 5, Name: "slow.zip", Size: 1000, DownloadURL: srv.URL}}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	s := newStore(t)
	if _, err := Fetch(ctx, client, s, files, func(Status) {}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: error = %v, want context.Canceled", err)
	}

	defer func(old time.Duration) { stallTimeout = old }(stallTimeout)
	stallTimeout = 50 * time.Millisecond
	if _, err := Fetch(context.Background(), client, s, files, func(Status) {}); !errors.Is(err, ErrStalled) {
		t.Errorf("stalled: error = %v, want ErrStalled", err)
	}
	if names := storeFiles(t, s); len(names) != 0 {
		t.Errorf("stopped fetches left files behind: %v", names)
	}
}
