package main

import (
	"errors"
	"testing"
	"time"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdata"
)

func TestRestoreDialog(t *testing.T) {
	modified := time.Unix(1_790_000_000, 0)
	backups := []appdata.SettingsBackup{
		{Path: "/data/Settings/Settings.toml.bkp-1", Slot: 1, Modified: modified},
		{Path: "/data/Settings/Settings.toml.bkp-2", Slot: 2, Err: errors.New("damaged")},
		{Path: "/data/Settings/Settings.toml.bkp-3", Slot: 3, Modified: modified},
	}

	mainthread.Wait(func() {
		var choices []int
		d := newRestoreDialog(errors.New("bad toml"), backups, "/data/Settings",
			func(choice int) { choices = append(choices, choice) })

		if got := d.Value("error").ToString(); got != "bad toml" {
			t.Errorf("error = %q", got)
		}
		if !d.Value("damaged").ToBool() {
			t.Error("damaged = false with a damaged backup")
		}
		list := d.Value("backups").ToList()
		if len(list) != 2 {
			t.Fatalf("listed %d backups, want the 2 that can be restored", len(list))
		}
		first := list[0].ToMap()
		slot, when, url := first["slot"], first["modified"], first["url"]
		if slot.ToInt() != 1 || when.ToLongLong() != modified.Unix() ||
			url.ToString() != "file:///data/Settings/Settings.toml.bkp-1" {
			t.Errorf("first backup = slot %d, modified %d, url %q", slot.ToInt(), when.ToLongLong(), url.ToString())
		}
		second := list[1].ToMap()["slot"]
		if got := second.ToInt(); got != 3 {
			t.Errorf("second backup's slot = %d, want 3", got)
		}
		if got := d.Value("settingsDirUrl").ToString(); got != "file:///data/Settings" {
			t.Errorf("settingsDirUrl = %q", got)
		}
		if len(choices) != 0 {
			t.Errorf("answered %v before QML did", choices)
		}
	})
}

func TestAskToRestoreSettingsWithoutBackups(t *testing.T) {
	damaged := []appdata.SettingsBackup{{Slot: 1, Err: errors.New("damaged")}}
	for _, backups := range [][]appdata.SettingsBackup{nil, damaged} {
		if got := mainthread.Wait2(func() int { return askToRestoreSettings(errors.New("bad toml"), backups, "/data") }); got != restoreIgnore {
			t.Errorf("with backups %+v: answer = %d, want restoreIgnore", backups, got)
		}
	}
}
