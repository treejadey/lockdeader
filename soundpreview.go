package main

import (
	"context"
	"log/slog"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/audio"
	"github.com/treejadey/lockdeader/banana"
)

// soundPreview plays sounds' audio previews for QML, one at a time. QML
// controls it by writing keys:
//
//	source: the preview's URL, which is downloaded and decoded; "" for none
//	state:  "playing", "paused", or "stopped" to go back to the start
//	seek:   a position to jump to, in milliseconds; also works while loading
//	volume: from 0 to 1, for the whole app
//	muted:  whether it's muted
//
// Go fills in "position" and "duration" (in milliseconds), "loading" and
// "error", and sets "state" back to "stopped" once a sound plays to its end.
// "durations" maps the URL of every sound loaded so far to its duration, so
// players for sounds that aren't the source can still show how long they are.
// "defaultVolume" is the volume to go back to when unmuting at no volume.
//
// Several players can show the same source; they all reflect these keys, so
// they stay in step with each other.
// All methods must be called on the Qt main thread.
type soundPreview struct {
	*qml.QQmlPropertyMap

	client    *banana.Client
	newEngine func() (*audio.Engine, error) // audio.NewEngine, except in tests
	engine    *audio.Engine                 // opened when the first sound loads
	sound     *audio.Sound                  // nil until the source has loaded

	source      string
	generation  int    // goes up with every source change
	state       string // as last asked for, even while still loading
	volume      float64
	muted       bool
	pendingSeek time.Duration // where to start once loaded; -1 for the start

	durations map[string]qt.QVariant // milliseconds, by URL

	onVolumeChange func(volume float64, muted bool)

	ticker *qt.QTimer
}

const positionInterval = 33 * time.Millisecond

func newSoundPreview(client *banana.Client, audioState appdb.AudioState, onVolumeChange func(volume float64, muted bool)) *soundPreview {
	p := &soundPreview{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		client:          client,
		newEngine:       audio.NewEngine,
		state:           "stopped",
		volume:          audioState.Volume,
		muted:           audioState.Muted,
		pendingSeek:     -1,
		durations:       make(map[string]qt.QVariant),
		onVolumeChange:  onVolumeChange,
		ticker:          qt.NewQTimer(),
	}

	p.ticker.SetInterval(int(positionInterval.Milliseconds()))
	p.ticker.OnTimeout(p.tick)

	p.Insert("source", qt.NewQVariant14(""))
	p.Insert("state", qt.NewQVariant14(p.state))
	p.Insert("seek", qt.NewQVariant4(-1))
	p.Insert("volume", qt.NewQVariant10(float32(p.volume)))
	p.Insert("muted", qt.NewQVariant8(p.muted))
	p.Insert("defaultVolume", qt.NewQVariant10(float32(appdb.DefaultAudioState().Volume)))
	p.Insert("durations", qt.NewQVariant20(p.durations))
	p.setPosition(0)
	p.Insert("duration", qt.NewQVariant4(0))
	p.Insert("loading", qt.NewQVariant8(false))
	p.setError(nil)

	// Only fires for changes made from QML, not for our own Inserts.
	p.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "source":
			p.Load(value.ToString())
		case "state":
			p.SetState(value.ToString())
		case "seek":
			if ms := value.ToInt(); ms >= 0 {
				// Clear the key, so seeking to the same place again is still a change.
				p.Insert("seek", qt.NewQVariant4(-1))
				p.Seek(time.Duration(ms) * time.Millisecond)
			}
		case "volume":
			p.SetVolume(value.ToDouble(), p.muted)
		case "muted":
			p.SetVolume(p.volume, value.ToBool())
		}
	})

	return p
}

// Load downloads and decodes in the background; "" just stops the current
// sound. If QML asks to play it meanwhile, it plays once it's loaded.
func (p *soundPreview) Load(url string) {
	if url == p.source {
		return
	}

	p.unload()
	p.source = url
	// Players compare their own sound with this, so it has to be right even
	// when Go changes it, e.g. stopping the sound for a new listing.
	p.Insert("source", qt.NewQVariant14(url))
	p.generation++

	if url == "" {
		return
	}

	if p.engine == nil {
		engine, err := p.newEngine()
		if err != nil {
			slog.Error("failed to open the audio output", "err", err)
			p.setError(err)

			return
		}

		p.engine = engine
	}

	p.Insert("loading", qt.NewQVariant8(true))
	generation, engine := p.generation, p.engine

	go func() {
		data, err := p.client.Download(context.Background(), url)
		var sound *audio.Sound

		if err == nil {
			sound, err = engine.Load(data)
		}

		mainthread.Start(func() {
			if generation != p.generation {
				// Another source was picked in the meantime.
				if sound != nil {
					sound.Close()
				}

				return
			}

			p.Insert("loading", qt.NewQVariant8(false))

			if err != nil {
				slog.Error("failed to load sound preview", "url", url, "err", err)
				p.pendingSeek = -1
				p.setError(err)

				return
			}

			p.sound = sound
			p.applyVolume()

			duration := int(sound.Length().Milliseconds())
			p.Insert("duration", qt.NewQVariant4(duration))
			p.durations[url] = *qt.NewQVariant4(duration)
			p.Insert("durations", qt.NewQVariant20(p.durations))

			if p.pendingSeek >= 0 {
				position := p.pendingSeek
				p.pendingSeek = -1
				p.Seek(position)
			}

			if p.state == "playing" {
				p.play()
			}
		})
	}()
}

func (p *soundPreview) SetState(state string) {
	switch state {
	case "playing", "paused", "stopped":
		p.state = state
	default:
		slog.Error("unknown sound preview state", "state", state)
	}

	if p.sound != nil {
		switch p.state {
		case "playing":
			p.play()
		case "paused":
			p.pause()
		case "stopped":
			p.stop()
		}
	}

	p.Insert("state", qt.NewQVariant14(p.state))
}

// Seek turns a stopped sound into a paused one, so playing it starts from
// position. While the sound is still loading, it starts there once loaded.
func (p *soundPreview) Seek(position time.Duration) {
	if p.sound == nil {
		if p.source != "" {
			p.pendingSeek = position
		}

		return
	}

	if err := p.sound.Seek(position); err != nil {
		p.fail(err)

		return
	}

	// Reported straight away: the sound only moves once it's next mixed.
	p.setPosition(position)

	if p.state == "stopped" {
		p.state = "paused"
		p.Insert("state", qt.NewQVariant14(p.state))
	}
}

func (p *soundPreview) play() {
	if err := p.engine.Start(); err != nil {
		p.fail(err)

		return
	}

	if err := p.sound.Play(); err != nil {
		p.fail(err)

		return
	}

	p.ticker.Start2()
}

func (p *soundPreview) pause() {
	p.ticker.Stop()

	if err := p.sound.Pause(); err != nil {
		p.fail(err)

		return
	}

	// Nothing else plays, so the device can rest.
	if err := p.engine.Stop(); err != nil {
		slog.Error("failed to stop the audio output", "err", err)
	}

	p.setPosition(p.sound.Position())
}

func (p *soundPreview) stop() {
	p.pause()

	if err := p.sound.Seek(0); err != nil {
		p.fail(err)

		return
	}

	p.setPosition(0)
}

func (p *soundPreview) tick() {
	if p.sound == nil {
		p.ticker.Stop()

		return
	}

	if p.sound.AtEnd() {
		p.state = "stopped"
		p.stop()
		p.Insert("state", qt.NewQVariant14(p.state))

		return
	}

	p.setPosition(p.sound.Position())
}

func (p *soundPreview) unload() {
	p.ticker.Stop()

	if p.sound != nil {
		p.sound.Close()
		p.sound = nil

		if err := p.engine.Stop(); err != nil {
			slog.Error("failed to stop the audio output", "err", err)
		}
	}

	p.state = "stopped"
	p.pendingSeek = -1
	p.Insert("state", qt.NewQVariant14(p.state))
	p.setPosition(0)
	p.Insert("duration", qt.NewQVariant4(0))
	p.Insert("loading", qt.NewQVariant8(false))
	p.setError(nil)
}

func (p *soundPreview) SetVolume(volume float64, muted bool) {
	volume = max(0, min(1, volume))
	if volume == p.volume && muted == p.muted {
		return
	}

	p.volume, p.muted = volume, muted
	p.Insert("volume", qt.NewQVariant10(float32(volume)))
	p.Insert("muted", qt.NewQVariant8(muted))
	p.applyVolume()

	if p.onVolumeChange != nil {
		p.onVolumeChange(volume, muted)
	}
}

func (p *soundPreview) applyVolume() {
	if p.sound == nil {
		return
	}

	volume := p.volume

	if p.muted {
		volume = 0
	}

	p.sound.SetVolume(float32(volume))
}

func (p *soundPreview) fail(err error) {
	slog.Error("sound preview failed", "err", err)
	p.ticker.Stop()
	p.state = "stopped"
	p.Insert("state", qt.NewQVariant14(p.state))
	p.setError(err)
}

func (p *soundPreview) setPosition(position time.Duration) {
	p.Insert("position", qt.NewQVariant4(int(position.Milliseconds())))
}

func (p *soundPreview) setError(err error) {
	msg := ""

	if err != nil {
		msg = err.Error()
	}

	p.Insert("error", qt.NewQVariant14(msg))
}
