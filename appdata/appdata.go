// Package appdata manages lockdeader's data directory, which holds its
// settings and caches.
package appdata

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

const EnvDataDir = "LOCKDEADER_DATA_DIR"

const appName = "Lockdeader"

// The data directory's layout:
//
//	Lockdeader/
//	    Lockdeader.db  the database: app state and installed mods (see appdb)
//	    Settings/
//	        Settings.toml
//	    Cache/
//	        Images/    images shown in the app: thumbnails and previews
//	        Mods/      the VPKs taken out of installed mods, named by SHA-256 (see install)
type Dirs struct {
	Root       string
	Settings   string
	Cache      string
	ImageCache string
	ModCache   string
}

func Layout(root string) Dirs {
	cache := filepath.Join(root, "Cache")

	return Dirs{
		Root:       root,
		Settings:   filepath.Join(root, "Settings"),
		Cache:      cache,
		ImageCache: filepath.Join(cache, "Images"),
		ModCache:   filepath.Join(cache, "Mods"),
	}
}

func (d Dirs) DatabaseFile() string {
	return filepath.Join(d.Root, "Lockdeader.db")
}

func (d Dirs) SettingsFile() string {
	return filepath.Join(d.Settings, "Settings.toml")
}

// GameInfoBackup is where the game's gameinfo.gi is copied before
// Lockdeader first writes to it.
func (d Dirs) GameInfoBackup() string {
	return filepath.Join(d.Root, "gameinfo.gi.bak")
}

func (d Dirs) Create() error {
	for _, dir := range []string{d.Root, d.Settings, d.Cache, d.ImageCache, d.ModCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("appdata: creating %s: %w", dir, err)
		}
	}

	return nil
}

func Setup() (Dirs, error) {
	root, err := Locate()
	if err != nil {
		return Dirs{}, err
	}

	dirs := Layout(root)

	return dirs, dirs.Create()
}

// The data directory goes in:
//
//   - Linux and other Unix-likes: $XDG_DATA_HOME/Lockdeader, which is
//     ~/.local/share/Lockdeader by default
//   - Windows: %LOCALAPPDATA%\Lockdeader. Local rather than roaming
//     (%APPDATA%), since caches and mods are large and machine-specific.
//   - macOS: ~/Library/Application Support/Lockdeader
//
// EnvDataDir overrides all of these.
func Locate() (string, error) {
	return locate(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func locate(goos string, getenv func(string) string, homeDir func() (string, error)) (string, error) {
	if dir := getenv(EnvDataDir); dir != "" {
		return dir, nil
	}

	switch goos {
	case "windows":
		dir := getenv("LOCALAPPDATA")
		if dir == "" {
			return "", errors.New("appdata: %LOCALAPPDATA% is not set")
		}

		return filepath.Join(dir, appName), nil

	case "darwin", "ios":
		home, err := homeDir()
		if err != nil {
			return "", fmt.Errorf("appdata: %w", err)
		}

		return filepath.Join(home, "Library", "Application Support", appName), nil

	default:
		// The XDG spec says to ignore relative paths. Its paths are POSIX,
		// so a leading slash is what counts, whatever the host (the tests
		// run this branch on Windows too).
		if dir := getenv("XDG_DATA_HOME"); path.IsAbs(dir) {
			return filepath.Join(dir, appName), nil
		}

		home, err := homeDir()
		if err != nil {
			return "", fmt.Errorf("appdata: %w", err)
		}

		return filepath.Join(home, ".local", "share", appName), nil
	}
}
