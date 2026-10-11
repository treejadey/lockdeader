package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/audio"
	"github.com/treejadey/lockdeader/banana"
)

// newTestPreview's engine has no device. Its server serves the 0.5 s test tone
// at /tone.wav, nothing at /missing.wav, and holds requests to /slow.wav until
// release is closed.
func newTestPreview(t *testing.T) (p *soundPreview, srv *httptest.Server, release chan struct{}) {
	t.Helper()

	tone, err := os.ReadFile("audio/testdata/tone.wav")
	if err != nil {
		t.Fatal(err)
	}

	release = make(chan struct{})

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tone.wav":
			w.Write(tone)
		case "/slow.wav":
			<-release
			w.Write(tone)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := banana.NewClient()
	client.HTTPClient = srv.Client()

	mainthread.Wait(func() {
		p = newSoundPreview(client, appdb.DefaultAudioState(), nil)
		p.newEngine = audio.NewEngineWithoutDevice
	})

	t.Cleanup(func() { mainthread.Wait(func() { p.Load("") }) })

	return p, srv, release
}

func value[T any](p *soundPreview, key string, convert func(p *soundPreview, key string) T) T {
	return mainthread.Wait2(func() T { return convert(p, key) })
}

func intValue(p *soundPreview, key string) int       { return p.Value(key).ToInt() }
func stringValue(p *soundPreview, key string) string { return p.Value(key).ToString() }

// waitFor checks cond on the main thread.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for !mainthread.Wait2(cond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}

		time.Sleep(5 * time.Millisecond)
	}
}

// advance stands in for the audio device and the ticker, for d's worth of sound.
func advance(t *testing.T, p *soundPreview, d time.Duration) {
	t.Helper()

	mainthread.Wait(func() {
		if _, err := p.engine.Read(int(d.Seconds() * 48000)); err != nil {
			t.Error(err)
		}

		p.tick()
	})
}

func near(got, want int) bool {
	return got >= want-30 && got <= want+30
}

func TestSoundPreviewPlayback(t *testing.T) {
	p, srv, _ := newTestPreview(t)

	mainthread.Wait(func() { p.Load(srv.URL + "/tone.wav") })
	waitFor(t, "the sound to load", func() bool { return p.sound != nil })

	if got := value(p, "duration", intValue); !near(got, 500) {
		t.Errorf("duration = %d, want about 500", got)
	}

	mainthread.Wait(func() { p.SetState("playing") })
	advance(t, p, 200*time.Millisecond)

	if got := value(p, "position", intValue); !near(got, 200) {
		t.Errorf("position after 200ms = %d", got)
	}

	mainthread.Wait(func() { p.SetState("paused") })
	advance(t, p, 100*time.Millisecond)

	if got := value(p, "position", intValue); !near(got, 200) {
		t.Errorf("position after pausing = %d, want it unchanged", got)
	}

	mainthread.Wait(func() { p.SetState("playing") })
	advance(t, p, 400*time.Millisecond)

	if got := value(p, "state", stringValue); got != "stopped" {
		t.Errorf("state after the end = %q, want stopped", got)
	}

	if got := value(p, "position", intValue); got != 0 {
		t.Errorf("position after the end = %d, want 0", got)
	}

	// Seeking a stopped sound pauses it there, and playing starts from there.
	mainthread.Wait(func() { p.Seek(300 * time.Millisecond) })

	if got := value(p, "state", stringValue); got != "paused" {
		t.Errorf("state after seeking = %q, want paused", got)
	}

	mainthread.Wait(func() { p.SetState("playing") })
	advance(t, p, 100*time.Millisecond)

	if got := value(p, "position", intValue); !near(got, 400) {
		t.Errorf("position after seeking to 300ms and playing 100ms = %d", got)
	}
}

func TestSoundPreviewPlayWhileLoading(t *testing.T) {
	p, srv, release := newTestPreview(t)

	mainthread.Wait(func() {
		p.Load(srv.URL + "/slow.wav")
		p.SetState("playing")
	})

	if got := value(p, "state", stringValue); got != "playing" {
		t.Errorf("state while loading = %q, want playing", got)
	}

	close(release)
	waitFor(t, "the sound to load", func() bool { return p.sound != nil })

	advance(t, p, 100*time.Millisecond)

	if got := value(p, "position", intValue); !near(got, 100) {
		t.Errorf("position = %d, want the sound to have played once loaded", got)
	}
}

func TestSoundPreviewSourceChangeWhileLoading(t *testing.T) {
	p, srv, release := newTestPreview(t)

	mainthread.Wait(func() {
		p.Load(srv.URL + "/slow.wav")
		p.Load("")
	})

	// Let the abandoned download finish and be dropped.
	close(release)
	time.Sleep(200 * time.Millisecond)

	if mainthread.Wait2(func() bool { return p.sound != nil }) {
		t.Error("the abandoned sound was loaded")
	}

	if got := value(p, "loading", func(p *soundPreview, key string) bool { return p.Value(key).ToBool() }); got {
		t.Error("still loading")
	}

	if got := value(p, "source", stringValue); got != "" {
		t.Errorf("source = %q, want none", got)
	}
}

func TestSoundPreviewLoadError(t *testing.T) {
	p, srv, _ := newTestPreview(t)

	mainthread.Wait(func() { p.Load(srv.URL + "/missing.wav") })
	waitFor(t, "the error", func() bool { return p.Value("error").ToString() != "" })

	if mainthread.Wait2(func() bool { return p.sound != nil }) {
		t.Error("a sound was loaded")
	}
}

func TestSoundPreviewSeekWhileLoading(t *testing.T) {
	p, srv, release := newTestPreview(t)

	// What a player does when its seek bar is clicked before its sound is the
	// source: switch to it, seek, and play.
	mainthread.Wait(func() {
		p.Load(srv.URL + "/slow.wav")
		p.Seek(300 * time.Millisecond)
		p.SetState("playing")
	})

	close(release)
	waitFor(t, "the sound to load", func() bool { return p.sound != nil })

	advance(t, p, 100*time.Millisecond)

	if got := value(p, "position", intValue); !near(got, 400) {
		t.Errorf("position = %d, want it to have started at 300ms", got)
	}
}

func TestSoundPreviewDurations(t *testing.T) {
	p, srv, _ := newTestPreview(t)
	url := srv.URL + "/tone.wav"

	mainthread.Wait(func() { p.Load(url) })
	waitFor(t, "the sound to load", func() bool { return p.sound != nil })

	// Still known once another sound is the source.
	mainthread.Wait(func() { p.Load("") })

	durations := mainthread.Wait2(func() map[string]qt.QVariant { return p.Value("durations").ToMap() })
	ms, ok := durations[url]
	if !ok {
		t.Fatalf("durations = %v, want %s in it", durations, url)
	}

	if got := ms.ToInt(); !near(got, 500) {
		t.Errorf("duration = %d, want about 500", got)
	}
}

func TestSoundPreviewVolume(t *testing.T) {
	p, _, _ := newTestPreview(t)

	type change struct {
		volume float64
		muted  bool
	}

	var changes []change

	mainthread.Wait(func() {
		p.onVolumeChange = func(volume float64, muted bool) { changes = append(changes, change{volume, muted}) }

		p.SetVolume(0.5, false)
		p.SetVolume(0.5, false) // no change
		p.SetVolume(2, true)
	})

	want := []change{{0.5, false}, {1, true}}
	if len(changes) != len(want) || changes[0] != want[0] || changes[1] != want[1] {
		t.Errorf("changes = %v, want %v", changes, want)
	}

	if got := mainthread.Wait2(func() float64 { return p.Value("volume").ToDouble() }); got != 1 {
		t.Errorf("volume = %g, want it clamped to 1", got)
	}
}
