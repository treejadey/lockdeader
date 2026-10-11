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

// SettingsBackups is how many backups of the settings file are kept.
const SettingsBackups = 10

// SettingsBackupFile is the path of a backup of the settings file at path,
// from slot 1 (the newest) to SettingsBackups (the oldest), e.g.
// Settings.toml.bkp-1.
func SettingsBackupFile(path string, slot int) string {
	return path + ".bkp-" + strconv.Itoa(slot)
}

// BackUpSettings keeps previous, the settings file's contents from before
// the app changed it, as a backup next to the file at path, if the file is no
// longer the same. It's meant to be called on quit, with what the file held
// at startup, so each backup is the settings from before a session that
// changed them.
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

// SettingsBackup is a backup of the settings file.
type SettingsBackup struct {
	Path     string
	Slot     int // see SettingsBackupFile
	Modified time.Time
	Err      error // why it can't be restored; nil if it can
}

// FindSettingsBackups returns the backups of the settings file at path, newest
// first, each checked to see whether it could be restored.
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

// BrokenSettingsFile is where RestoreSettingsBackup keeps the settings file it
// replaces, in case the user wants to fix it rather than lose it.
func BrokenSettingsFile(path string) string {
	return path + ".broken"
}

// RestoreSettingsBackup replaces the settings file at path with its backup in
// slot. The file it replaces is kept as BrokenSettingsFile, replacing any kept
// before. The backup stays where it is.
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
