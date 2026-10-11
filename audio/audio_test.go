package audio

import (
	"math"
	"os"
	"testing"
	"time"
)

// The engine without a device mixes at this rate.
const sampleRate = 48000

// The fixtures are a 0.5 s, 440 Hz tone at half volume.
const toneLength = 500 * time.Millisecond

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	e, err := NewEngineWithoutDevice()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(e.Close)

	return e
}

func loadFixture(t *testing.T, e *Engine, name string) *Sound {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}

	s, err := e.Load(data)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(s.Close)

	return s
}

// Mixes d's worth of frames, and returns the loudest sample.
func read(t *testing.T, e *Engine, d time.Duration) float64 {
	t.Helper()

	out, err := e.Read(int(d.Seconds() * sampleRate))
	if err != nil {
		t.Fatal(err)
	}

	var peak float64

	for _, v := range out {
		peak = math.Max(peak, math.Abs(float64(v)))
	}

	return peak
}

func near(got, want time.Duration) bool {
	return (got - want).Abs() <= 30*time.Millisecond
}

func TestFormats(t *testing.T) {
	for _, name := range []string{"tone.mp3", "tone.ogg", "tone.wav"} {
		t.Run(name, func(t *testing.T) {
			e := newTestEngine(t)
			s := loadFixture(t, e, name)

			if got := s.Length(); !near(got, toneLength) {
				t.Errorf("Length() = %v, want about %v", got, toneLength)
			}

			if err := s.Play(); err != nil {
				t.Fatal(err)
			}

			if peak := read(t, e, 100*time.Millisecond); peak < 0.2 {
				t.Errorf("peak = %.2f, want the tone to be heard", peak)
			}
		})
	}
}

func TestPlayback(t *testing.T) {
	e := newTestEngine(t)
	s := loadFixture(t, e, "tone.wav")

	// Paused sounds stay put and silent.
	if peak := read(t, e, 100*time.Millisecond); peak != 0 {
		t.Errorf("peak while paused = %.2f, want silence", peak)
	}

	if got := s.Position(); got != 0 {
		t.Errorf("Position() before playing = %v, want 0", got)
	}

	if err := s.Play(); err != nil {
		t.Fatal(err)
	}

	read(t, e, 200*time.Millisecond)

	if got := s.Position(); !near(got, 200*time.Millisecond) {
		t.Errorf("Position() after 200ms = %v", got)
	}

	if err := s.Pause(); err != nil {
		t.Fatal(err)
	}

	read(t, e, 100*time.Millisecond)

	if got := s.Position(); !near(got, 200*time.Millisecond) {
		t.Errorf("Position() after pausing = %v, want it unchanged", got)
	}

	if err := s.Seek(100 * time.Millisecond); err != nil {
		t.Fatal(err)
	}

	if err := s.Play(); err != nil {
		t.Fatal(err)
	}

	read(t, e, 100*time.Millisecond)

	if got := s.Position(); !near(got, 200*time.Millisecond) {
		t.Errorf("Position() after seeking to 100ms and playing 100ms = %v", got)
	}

	if s.AtEnd() {
		t.Error("AtEnd() before the end")
	}

	read(t, e, 400*time.Millisecond)

	if !s.AtEnd() {
		t.Error("AtEnd() = false after playing past the end")
	}

	if peak := read(t, e, 100*time.Millisecond); peak != 0 {
		t.Errorf("peak after the end = %.2f, want silence", peak)
	}
}

func TestVolume(t *testing.T) {
	e := newTestEngine(t)
	s := loadFixture(t, e, "tone.wav")
	s.SetVolume(0.5)

	if err := s.Play(); err != nil {
		t.Fatal(err)
	}

	// The tone peaks at 0.5, so 0.25 at half volume.
	if peak := read(t, e, 100*time.Millisecond); math.Abs(peak-0.25) > 0.02 {
		t.Errorf("peak at half volume = %.3f, want about 0.25", peak)
	}
}

func TestLoadInvalid(t *testing.T) {
	e := newTestEngine(t)

	for name, data := range map[string][]byte{
		"empty":   nil,
		"garbage": []byte("<html>not a sound</html>"),
	} {
		if s, err := e.Load(data); err == nil {
			s.Close()
			t.Errorf("Load(%s) succeeded, want an error", name)
		}
	}
}
