package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/network"

	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/i18n"
)

// Qt has to be used from the thread that created it, but Go runs each test on
// a goroutine of its own. So the main thread runs Qt's event loop while the
// tests run in the background, handing their Qt calls over with mainthread.
func init() {
	runtime.LockOSThread()
}

func TestMain(m *testing.M) {
	// The clipboard needs a Qt application; the offscreen platform has an
	// in-process clipboard and needs no display.
	os.Setenv("QT_QPA_PLATFORM", "offscreen")
	qt.NewQGuiApplication([]string{"test"})

	// As in the app, which English needs for plurals; in English whatever
	// the system's language, as tests expect.
	english, err := i18n.Load(translationFS(), []string{"en"})
	if err != nil {
		panic(err)
	}

	installTranslator(english)

	if err := registerQML(); err != nil {
		panic(err)
	}

	code := 0

	go func() {
		code = m.Run()
		mainthread.Start(qt.QCoreApplication_Exit)
	}()

	qt.QGuiApplication_Exec()
	os.Exit(code)
}

// waitForMessage waits for the bridge's id'th copy.
func waitForMessage(t *testing.T, b *clipboardBridge, id int) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for mainthread.Wait2(func() int { return b.messages }) < id {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the copy to finish")
		}

		time.Sleep(5 * time.Millisecond)
	}

	return mainthread.Wait2(func() string { return b.Value("message").ToString() })
}

func clipboardText() string {
	return mainthread.Wait2(func() string { return qt.QGuiApplication_Clipboard().Text() })
}

// clipboardImageSize returns 0x0 when there's no image.
func clipboardImageSize() (width, height int) {
	mainthread.Wait(func() {
		img := qt.QGuiApplication_Clipboard().Image()
		width, height = img.Width(), img.Height()
	})

	return width, height
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()

	var buf bytes.Buffer

	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func newTestBridge(t *testing.T) *clipboardBridge {
	t.Helper()

	dir := t.TempDir()

	return mainthread.Wait2(func() *clipboardBridge {
		qt.QGuiApplication_Clipboard().Clear()

		return newClipboardBridge(dir, banana.NewClient())
	})
}

func TestCopyLink(t *testing.T) {
	const url = "https://images.gamebanana.com/img/ss/mods/example.jpg"

	b := newTestBridge(t)
	mainthread.Wait(func() { b.CopyLink(url) })

	if got := clipboardText(); got != url {
		t.Errorf("clipboard text = %q, want %q", got, url)
	}

	if got := waitForMessage(t, b, 1); got != "Link copied" {
		t.Errorf("message = %q", got)
	}

	if got := mainthread.Wait2(func() int { return b.Value("messageId").ToInt() }); got != 1 {
		t.Errorf("messageId = %d, want 1", got)
	}
}

func TestCopyImageDownloads(t *testing.T) {
	var requests atomic.Int32
	img := pngBytes(t, 40, 30)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)

		if r.UserAgent() == "" {
			t.Error("download sent no User-Agent")
		}

		w.Write(img)
	}))
	defer srv.Close()

	b := newTestBridge(t)
	mainthread.Wait(func() { b.CopyImage(srv.URL + "/image.png") })

	if got := waitForMessage(t, b, 1); got != "Image copied" {
		t.Fatalf("message = %q", got)
	}

	if w, h := clipboardImageSize(); w != 40 || h != 30 {
		t.Errorf("clipboard image is %dx%d, want 40x30", w, h)
	}

	if requests.Load() != 1 {
		t.Errorf("server got %d requests, want 1", requests.Load())
	}
}

func TestCopyImageFromCache(t *testing.T) {
	data := pngBytes(t, 12, 34)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}))
	defer srv.Close()

	b := newTestBridge(t)
	url := srv.URL + "/cached.png"

	// The image gets into the cache as QML's images do, by Qt fetching it.
	// Writing the entry directly needs QNetworkCacheMetaData.SetRawHeaders
	// (Qt ignores entries without headers), which in miqt v0.14.0 writes past
	// the end of its buffer and corrupts the heap. The manager owns the cache
	// from here on, so it's never deleted.
	reply := mainthread.Wait2(func() *network.QNetworkReply {
		manager := network.NewQNetworkAccessManager()
		manager.SetCache(b.cache.QAbstractNetworkCache)

		return manager.Get(network.NewQNetworkRequest2(qt.NewQUrl3(url)))
	})

	deadline := time.Now().Add(5 * time.Second)

	for !mainthread.Wait2(reply.IsFinished) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for Qt to fetch the image")
		}

		time.Sleep(5 * time.Millisecond)
	}

	cached := mainthread.Wait2(func() bool {
		reply.DeleteLater()

		return reply.Error() == network.QNetworkReply__NoError && b.cache.MetaData(qt.NewQUrl3(url)).IsValid()
	})

	if !cached {
		t.Fatal("couldn't add the image to the cache")
	}

	// With the server gone, this can only work from the cache.
	srv.Close()
	mainthread.Wait(func() { b.CopyImage(url) })

	if got := waitForMessage(t, b, 1); got != "Image copied" {
		t.Fatalf("message = %q", got)
	}

	if w, h := clipboardImageSize(); w != 12 || h != 34 {
		t.Errorf("clipboard image is %dx%d, want 12x34", w, h)
	}
}

func TestCopyImageFailures(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"not found": func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
		"not an image": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html>definitely not a picture</html>"))
		},
	}

	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()

			b := newTestBridge(t)
			mainthread.Wait(func() { b.CopyImage(srv.URL + "/image.png") })

			if got := waitForMessage(t, b, 1); got != "Couldn’t copy the image" {
				t.Errorf("message = %q", got)
			}

			if w, h := clipboardImageSize(); w != 0 || h != 0 {
				t.Errorf("clipboard has a %dx%d image anyway", w, h)
			}
		})
	}
}
