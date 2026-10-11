package appdata

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestBackUpSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Settings.toml")

	// A session that didn't change anything leaves no backup.
	writeFile(t, path, "session 0")

	if err := BackUpSettings(path, []byte("session 0")); err != nil {
		t.Fatal(err)
	}

	if matches, _ := filepath.Glob(path + ".bkp-*"); len(matches) != 0 {
		t.Fatalf("backed up an unchanged file: %v", matches)
	}

	// Each session that does keeps what the file held before it, newest
	// first, up to SettingsBackups of them.
	for session := 1; session <= SettingsBackups+2; session++ {
		before := "session " + strconv.Itoa(session-1)
		writeFile(t, path, "session "+strconv.Itoa(session))

		if err := BackUpSettings(path, []byte(before)); err != nil {
			t.Fatal(err)
		}
	}

	for slot := 1; slot <= SettingsBackups; slot++ {
		data, err := os.ReadFile(SettingsBackupFile(path, slot))
		if err != nil {
			t.Fatal(err)
		}

		if want := "session " + strconv.Itoa(SettingsBackups+2-slot); string(data) != want {
			t.Errorf("backup %d = %q, want %q", slot, data, want)
		}
	}

	if _, err := os.Stat(SettingsBackupFile(path, SettingsBackups+1)); err == nil {
		t.Errorf("kept more than %d backups", SettingsBackups)
	}

	if data, _ := os.ReadFile(path); string(data) != "session "+strconv.Itoa(SettingsBackups+2) {
		t.Errorf("the settings file changed to %q", data)
	}

	if matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestBackUpSettingsWithoutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")

	if err := BackUpSettings(path, []byte("before")); err == nil {
		t.Error("want an error")
	}

	if _, err := os.Stat(SettingsBackupFile(path, 1)); err == nil {
		t.Error("made a backup with no settings file")
	}
}

func TestFindAndRestoreSettingsBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")

	if got := FindSettingsBackups(path); len(got) != 0 {
		t.Fatalf("backups with none made = %+v", got)
	}

	writeFile(t, path, "[cache\nbroken by hand")
	writeFile(t, SettingsBackupFile(path, 1), "[cache]\nimages_max_size_mb = 0\n") // invalid
	writeFile(t, SettingsBackupFile(path, 3), "[cache]\nimages_max_size_mb = 600\n")

	backups := FindSettingsBackups(path)
	if len(backups) != 2 || backups[0].Slot != 1 || backups[1].Slot != 3 {
		t.Fatalf("backups = %+v, want slots 1 and 3", backups)
	}

	if backups[0].Err == nil {
		t.Error("an invalid backup has no error")
	}

	if backups[1].Err != nil || backups[1].Modified.IsZero() {
		t.Errorf("a good backup = %+v", backups[1])
	}

	if err := RestoreSettingsBackup(path, 3); err != nil {
		t.Fatal(err)
	}

	s, err := LoadSettings(path)
	if err != nil || s.Cache.ImagesMaxSizeMB != 600 {
		t.Errorf("restored settings = %+v, %v", s, err)
	}

	if data, _ := os.ReadFile(BrokenSettingsFile(path)); string(data) != "[cache\nbroken by hand" {
		t.Errorf("the broken file was kept as %q", data)
	}

	if _, err := os.Stat(SettingsBackupFile(path, 3)); err != nil {
		t.Errorf("the restored backup is gone: %v", err)
	}
}
