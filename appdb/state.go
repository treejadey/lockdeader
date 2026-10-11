package appdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Keys in the app_state table.
const (
	keyVolume = "audio.volume"
	keyMuted  = "audio.muted"
)

type AudioState struct {
	Volume float64 // from 0 to 1
	Muted  bool
}

func DefaultAudioState() AudioState {
	// Low, because sounds on GameBanana tend to be loud.
	return AudioState{Volume: 0.35}
}

// Anything not stored yet, or stored out of range, gets its default.
func (d *DB) AudioState(ctx context.Context) (AudioState, error) {
	s := DefaultAudioState()
	var volume float64

	if ok, err := d.state(ctx, keyVolume, &volume); err != nil {
		return DefaultAudioState(), err
	} else if ok && volume >= 0 && volume <= 1 {
		s.Volume = volume
	}

	if _, err := d.state(ctx, keyMuted, &s.Muted); err != nil {
		return DefaultAudioState(), err
	}

	return s, nil
}

func (d *DB) SetAudioState(ctx context.Context, s AudioState) error {
	if s.Volume < 0 || s.Volume > 1 {
		return fmt.Errorf("appdb: volume must be from 0 to 1, not %g", s.Volume)
	}

	return d.setStates(ctx, map[string]any{keyVolume: s.Volume, keyMuted: s.Muted})
}

// ImportAudioState stores s only if there's no audio state stored already,
// and reports whether it did. It's for moving the audio state from where older
// versions kept it (the settings file) without overwriting newer state.
func (d *DB) ImportAudioState(ctx context.Context, s AudioState) (bool, error) {
	var volume float64

	if ok, err := d.state(ctx, keyVolume, &volume); err != nil || ok {
		return false, err
	}

	return true, d.SetAudioState(ctx, s)
}

// The bool reports whether there's a value stored under key.
func (d *DB) state(ctx context.Context, key string, dest any) (bool, error) {
	err := d.db.QueryRowContext(ctx, "SELECT value FROM app_state WHERE key = ?", key).Scan(dest)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("appdb: reading %s: %w", key, err)
	}

	return true, nil
}

// The values are stored all at once, in one transaction.
func (d *DB) setStates(ctx context.Context, values map[string]any) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	defer tx.Rollback() // no-op once committed

	for key, value := range values {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO app_state (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value",
			key, value)
		if err != nil {
			return fmt.Errorf("appdb: storing %s: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	return nil
}
