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

// The device only runs between Start and Stop.
type Engine struct {
	e        *C.ld_engine
	noDevice bool
}

// Opens the default output device, but doesn't start it.
func NewEngine() (*Engine, error) {
	return newEngine(false)
}

// For tests: the engine has no output device, and only mixes when Read is
// called.
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

// The engine's sounds must be closed first.
func (e *Engine) Close() {
	C.ld_engine_free(e.e)
}

// Without a device, it does nothing.
func (e *Engine) Start() error {
	if e.noDevice {
		return nil
	}

	return check("starting the output device", C.ld_engine_start(e.e))
}

// Stopping the device also stops time for every sound. Without a device, it
// does nothing.
func (e *Engine) Stop() error {
	if e.noDevice {
		return nil
	}

	return check("stopping the output device", C.ld_engine_stop(e.e))
}

// Mixes frames stereo frames at 48 kHz, advancing every playing sound. Only
// for engines without a device.
func (e *Engine) Read(frames int) ([]float32, error) {
	if !e.noDevice {
		return nil, errors.New("audio: mixing: the engine has a device")
	}

	out := make([]float32, frames*2)
	err := check("mixing", C.ld_engine_read(e.e, (*C.float)(&out[0]), C.ulonglong(frames)))

	return out, err
}

// data is an MP3, Ogg Vorbis, FLAC or WAV file's contents. The sound starts
// out paused at its beginning.
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

// Belongs to the engine that loaded it.
type Sound struct {
	s *C.ld_sound
}

func (s *Sound) Close() {
	C.ld_sound_free(s.s)
}

// Plays from the sound's position, and stops by itself at the end.
func (s *Sound) Play() error {
	return check("playing", C.ld_sound_play(s.s))
}

// Keeps the sound's position.
func (s *Sound) Pause() error {
	return check("pausing", C.ld_sound_pause(s.s))
}

func (s *Sound) Seek(position time.Duration) error {
	return check("seeking", C.ld_sound_seek(s.s, C.double(position.Seconds())))
}

func (s *Sound) Position() time.Duration {
	return seconds(C.ld_sound_position(s.s))
}

func (s *Sound) Length() time.Duration {
	return seconds(C.ld_sound_length(s.s))
}

func (s *Sound) AtEnd() bool {
	return C.ld_sound_at_end(s.s) != 0
}

// From 0 (silent) to 1 (as recorded).
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
