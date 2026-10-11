package appdata

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocate(t *testing.T) {
	home := func() (string, error) { return "/home/someone", nil }
	noHome := func() (string, error) { return "", errors.New("no home") }

	tests := []struct {
		name    string
		goos    string
		env     map[string]string
		homeDir func() (string, error)
		want    string
		wantErr bool
	}{
		{name: "linux default", goos: "linux", homeDir: home,
			want: filepath.Join("/home/someone", ".local", "share", "Lockdeader")},
		{name: "linux XDG_DATA_HOME", goos: "linux", homeDir: home,
			env:  map[string]string{"XDG_DATA_HOME": "/data"},
			want: filepath.Join("/data", "Lockdeader")},
		{name: "linux relative XDG_DATA_HOME is ignored", goos: "linux", homeDir: home,
			env:  map[string]string{"XDG_DATA_HOME": "data"},
			want: filepath.Join("/home/someone", ".local", "share", "Lockdeader")},
		{name: "freebsd behaves like linux", goos: "freebsd", homeDir: home,
			want: filepath.Join("/home/someone", ".local", "share", "Lockdeader")},
		{name: "windows", goos: "windows", homeDir: home,
			env:  map[string]string{"LOCALAPPDATA": `C:\Users\someone\AppData\Local`, "APPDATA": `C:\Users\someone\AppData\Roaming`},
			want: filepath.Join(`C:\Users\someone\AppData\Local`, "Lockdeader")},
		{name: "windows without LOCALAPPDATA", goos: "windows", homeDir: home, wantErr: true},
		{name: "macos", goos: "darwin", homeDir: home,
			want: filepath.Join("/home/someone", "Library", "Application Support", "Lockdeader")},
		{name: "no home directory", goos: "linux", homeDir: noHome, wantErr: true},
		{name: "override wins everywhere", goos: "windows", homeDir: noHome,
			env:  map[string]string{EnvDataDir: "/tmp/lockdeader-dev"},
			want: "/tmp/lockdeader-dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			got, err := locate(tt.goos, getenv, tt.homeDir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("locate() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("locate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetupCreatesLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Lockdeader")
	t.Setenv(EnvDataDir, root)

	dirs, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"Settings", "Cache", filepath.Join("Cache", "Images"), filepath.Join("Cache", "Mods")} {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil || !info.IsDir() {
			t.Errorf("%s: not created (%v)", rel, err)
		}
	}
	if got, want := dirs.SettingsFile(), filepath.Join(root, "Settings", "Settings.toml"); got != want {
		t.Errorf("SettingsFile() = %q, want %q", got, want)
	}

	// Running it again on an existing directory is fine.
	if _, err := Setup(); err != nil {
		t.Errorf("second Setup: %v", err)
	}
}

func TestLoadSettingsCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")

	s, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if s != DefaultSettings() {
		t.Errorf("settings = %+v, want defaults", s)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("file not created:", err)
	}
	text := string(data)
	for _, want := range []string{"# lockdeader settings.", "[browse]", "show_nsfw = false", "[cache]", "# How much disk space", "images_max_size_mb = 512"} {
		if !strings.Contains(text, want) {
			t.Errorf("file is missing %q:\n%s", want, text)
		}
	}

	// What was written reads back the same.
	again, err := LoadSettings(path)
	if err != nil || again != s {
		t.Errorf("reloaded = %+v, %v; want %+v", again, err, s)
	}

	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp"))
	if len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestLoadSettingsReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")
	writeFile(t, path, "[cache]\nimages_max_size_mb = 2048\n")

	s, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Cache.ImagesMaxSizeMB != 2048 {
		t.Errorf("ImagesMaxSizeMB = %d, want 2048", s.Cache.ImagesMaxSizeMB)
	}
}

func TestLoadSettingsKeepsDefaultsForMissingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")
	writeFile(t, path, "# nothing set yet\n")

	s, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if s != DefaultSettings() {
		t.Errorf("settings = %+v, want defaults", s)
	}
}

func TestLoadSettingsInvalidFile(t *testing.T) {
	tests := map[string]string{
		"not toml":     "[cache\nimages_max_size_mb = ",
		"wrong type":   "[cache]\nimages_max_size_mb = \"lots\"\n",
		"out of range": "[cache]\nimages_max_size_mb = 0\n",
		"negative":     "[cache]\nimages_max_size_mb = -5\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "Settings.toml")
			writeFile(t, path, content)

			s, err := LoadSettings(path)
			if err == nil {
				t.Fatal("want an error")
			}
			if s != DefaultSettings() {
				t.Errorf("settings = %+v, want defaults", s)
			}
			if data, _ := os.ReadFile(path); string(data) != content {
				t.Errorf("the user's file was changed to:\n%s", data)
			}
		})
	}
}

func TestSaveSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")
	s := DefaultSettings()
	s.Browse.ShowNSFW = true
	s.Cache.ImagesMaxSizeMB = 2048

	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil || got != s {
		t.Errorf("reloaded = %+v, %v; want %+v", got, err, s)
	}

	s.Cache.ImagesMaxSizeMB = 0
	if err := SaveSettings(path, s); err == nil {
		t.Error("saving an out-of-range cache size: want an error")
	}
	if got, _ := LoadSettings(path); got.Cache.ImagesMaxSizeMB != 2048 {
		t.Errorf("cache size = %d after a failed save, want 2048", got.Cache.ImagesMaxSizeMB)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadLegacyAudio(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Settings.toml")
	writeFile(t, path, "[browse]\nshow_nsfw = true\n\n[audio]\nvolume = 0.7\nmuted = true\n")

	got, ok, err := ReadLegacyAudio(path)
	if err != nil || !ok || got != (LegacyAudio{Volume: 0.7, Muted: true}) {
		t.Errorf("ReadLegacyAudio() = %+v, %v, %v", got, ok, err)
	}

	// The file still loads, and saving it drops the [audio] table.
	s, err := LoadSettings(path)
	if err != nil || !s.Browse.ShowNSFW {
		t.Fatalf("LoadSettings() = %+v, %v", s, err)
	}
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := ReadLegacyAudio(path); ok {
		t.Error("[audio] still there after saving")
	}

	for name, content := range map[string]string{
		"none":         "[browse]\nshow_nsfw = true\n",
		"out of range": "[audio]\nvolume = 3\n",
	} {
		writeFile(t, path, content)
		if got, ok, err := ReadLegacyAudio(path); err != nil || ok {
			t.Errorf("%s: ReadLegacyAudio() = %+v, %v, %v; want nothing", name, got, ok, err)
		}
	}
}
