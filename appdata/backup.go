package appdata

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"
)

const SettingsBackups = 10

// Slot 1 is the newest backup and slot SettingsBackups the oldest.
func SettingsBackupFile(path string, slot int) string {
	return path + ".bkp-" + strconv.Itoa(slot)
}

// previous is what the settings file held at startup; it's backed up only if
// the file has changed since. Call it on quit, so each backup is the settings
// from before a session that changed them.
//
// Like Chatterino's backups, the newest goes in slot 1, and the older ones
// move up a slot to make room for it; the one in the last slot is deleted.
func BackUpSettings(path string, previous []byte) error {
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	if bytes.Equal(current, previous) {
		return nil
	}

	if err := os.Remove(SettingsBackupFile(path, SettingsBackups)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("appdata: %w", err)
	}

	for slot := SettingsBackups - 1; slot >= 1; slot-- {
		err := os.Rename(SettingsBackupFile(path, slot), SettingsBackupFile(path, slot+1))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("appdata: %w", err)
		}
	}

	return writeAtomic(SettingsBackupFile(path, 1), previous)
}

type SettingsBackup struct {
	Path     string
	Slot     int // see SettingsBackupFile
	Modified time.Time
	Err      error // why it can't be restored; nil if it can
}

// The backups come newest first.
func FindSettingsBackups(path string) []SettingsBackup {
	var backups []SettingsBackup

	for slot := 1; slot <= SettingsBackups; slot++ {
		b := SettingsBackup{Path: SettingsBackupFile(path, slot), Slot: slot}

		info, err := os.Stat(b.Path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err == nil {
			b.Modified = info.ModTime()

			// A backup that doesn't load can't be restored.
			var data []byte

			if data, err = os.ReadFile(b.Path); err == nil {
				_, err = parseSettings(b.Path, data)
			}
		}

		b.Err = err
		backups = append(backups, b)
	}

	return backups
}

// RestoreSettingsBackup keeps the settings file it replaces here, so the user
// can fix it rather than lose it.
func BrokenSettingsFile(path string) string {
	return path + ".broken"
}

// The file it replaces is kept as BrokenSettingsFile, replacing any kept
// before, and the backup stays where it is.
func RestoreSettingsBackup(path string, slot int) error {
	data, err := os.ReadFile(SettingsBackupFile(path, slot))
	if err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	if err := os.Rename(path, BrokenSettingsFile(path)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("appdata: %w", err)
	}

	return writeAtomic(path, data)
}
