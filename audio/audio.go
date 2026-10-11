// Package audio plays sounds through the default output device, using the
// miniaudio library vendored in miniaudio/ (see its README).
//
// An Engine and its Sounds must only be used from one goroutine at a time,
// except that Load may run while other sounds play.
package audio

/*
#cgo linux LDFLAGS: -ldl -lpthread -lm
#cgo darwin LDFLAGS: -framework CoreFoundation -framework CoreAudio -framework AudioToolbox
#include "audio.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

// Engine mixes sounds and plays them through the default output device. The
// device only runs between Start and Stop.
type Engine struct {
	e        *C.ld_engine
	noDevice bool
}

// NewEngine opens the default output device, without starting it.
func NewEngine() (*Engine, error) {
	return newEngine(false)
}

// NewEngineWithoutDevice returns an engine with no output device, which only
// mixes when Read is called. It's for testing.
func NewEngineWithoutDevice() (*Engine, error) {
	return newEngine(true)
}

func newEngine(noDevice bool) (*Engine, error) {
	var e *C.ld_engine
	if err := check("opening the output device", C.ld_engine_new(cBool(noDevice), &e)); err != nil {
		return nil, err
	}
	return &Engine{e, noDevice}, nil
}

// Close closes the device. The engine's sounds must be closed first.
func (e *Engine) Close() {
	C.ld_engine_free(e.e)
}

// Start starts the output device, so playing sounds are heard. Without a
// device, it does nothing.
func (e *Engine) Start() error {
	if e.noDevice {
		return nil
	}
	return check("starting the output device", C.ld_engine_start(e.e))
}

// Stop stops the output device, which also stops time for every sound.
// Without a device, it does nothing.
func (e *Engine) Stop() error {
	if e.noDevice {
		return nil
	}
	return check("stopping the output device", C.ld_engine_stop(e.e))
}

// Read mixes frames stereo frames at 48 kHz, advancing every playing sound,
// for engines without a device.
func (e *Engine) Read(frames int) ([]float32, error) {
	if !e.noDevice {
		return nil, errors.New("audio: mixing: the engine has a device")
	}
	out := make([]float32, frames*2)
	err := check("mixing", C.ld_engine_read(e.e, (*C.float)(&out[0]), C.ulonglong(frames)))
	return out, err
}

// Load decodes a sound from an MP3, Ogg Vorbis, FLAC or WAV file's contents.
// The sound starts out paused at its beginning.
func (e *Engine) Load(data []byte) (*Sound, error) {
	if len(data) == 0 {
		return nil, errors.New("audio: decoding: no data")
	}
	var s *C.ld_sound
	if err := check("decoding", C.ld_sound_new(e.e, unsafe.Pointer(&data[0]), C.size_t(len(data)), &s)); err != nil {
		return nil, err
	}
	return &Sound{s}, nil
}

// Sound is a decoded sound, attached to the engine that loaded it.
type Sound struct {
	s *C.ld_sound
}

// Close stops the sound and frees it.
func (s *Sound) Close() {
	C.ld_sound_free(s.s)
}

// Play plays the sound from its position. It stops by itself at the end.
func (s *Sound) Play() error {
	return check("playing", C.ld_sound_play(s.s))
}

// Pause stops the sound, keeping its position.
func (s *Sound) Pause() error {
	return check("pausing", C.ld_sound_pause(s.s))
}

// Seek moves the sound's position.
func (s *Sound) Seek(position time.Duration) error {
	return check("seeking", C.ld_sound_seek(s.s, C.double(position.Seconds())))
}

// Position returns how far into the sound playback is.
func (s *Sound) Position() time.Duration {
	return seconds(C.ld_sound_position(s.s))
}

// Length returns how long the sound is.
func (s *Sound) Length() time.Duration {
	return seconds(C.ld_sound_length(s.s))
}

// AtEnd reports whether the sound has played to its end.
func (s *Sound) AtEnd() bool {
	return C.ld_sound_at_end(s.s) != 0
}

// SetVolume sets the sound's volume, from 0 (silent) to 1 (as recorded).
func (s *Sound) SetVolume(volume float32) {
	C.ld_sound_set_volume(s.s, C.float(volume))
}

func check(op string, result C.int) error {
	if result == 0 {
		return nil
	}
	return fmt.Errorf("audio: %s: %s", op, C.GoString(C.ld_result_string(result)))
}

func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func seconds(s C.double) time.Duration {
	return time.Duration(float64(s) * float64(time.Second))
}
