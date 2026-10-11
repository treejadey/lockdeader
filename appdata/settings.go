package appdata

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// App state that the user doesn't set as such, like the sound preview volume,
// is in the database (see appdb) instead.
type Settings struct {
	Browse BrowseSettings `toml:"browse"`
	Cache  CacheSettings  `toml:"cache"`
}

type BrowseSettings struct {
	ShowNSFW bool `toml:"show_nsfw" comment:"Whether mods GameBanana marks as not safe for work are listed."`
}

type CacheSettings struct {
	ImagesMaxSizeMB int `toml:"images_max_size_mb" comment:"How much disk space cached images (thumbnails and previews) may take up, in megabytes."`
}

func DefaultSettings() Settings {
	return Settings{
		Cache: CacheSettings{
			ImagesMaxSizeMB: 512,
		},
	}
}

func (s Settings) validate() error {
	if s.Cache.ImagesMaxSizeMB < 1 {
		return fmt.Errorf("cache.images_max_size_mb must be at least 1, not %d", s.Cache.ImagesMaxSizeMB)
	}

	return nil
}

const settingsHeader = `# lockdeader settings.
# Anything left out of this file uses its default value.

`

// The file is created with the default settings if it doesn't exist, and
// settings missing from it keep their defaults.
//
// If the file can't be read or is invalid, LoadSettings returns the defaults
// along with the error. It never overwrites an existing file.
func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		s := DefaultSettings()

		return s, writeSettings(path, s)
	}

	if err != nil {
		return DefaultSettings(), fmt.Errorf("appdata: %w", err)
	}

	return parseSettings(path, data)
}

// path is only used in errors. If data is invalid, parseSettings returns the
// defaults along with the error.
func parseSettings(path string, data []byte) (Settings, error) {
	s := DefaultSettings()

	if err := toml.Unmarshal(data, &s); err != nil {
		return DefaultSettings(), fmt.Errorf("appdata: reading %s: %w", path, err)
	}

	if err := s.validate(); err != nil {
		return DefaultSettings(), fmt.Errorf("appdata: %s: %w", path, err)
	}

	return s, nil
}

func SaveSettings(path string, s Settings) error {
	if err := s.validate(); err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	return writeSettings(path, s)
}

func writeSettings(path string, s Settings) error {
	data, err := toml.Marshal(s)
	if err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	return writeAtomic(path, append([]byte(settingsHeader), data...))
}

// This goes through a temporary file so that a crash halfway through can't
// leave a truncated file behind.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	defer os.Remove(tmp.Name()) // no-op once renamed

	_, err = tmp.Write(data)

	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return fmt.Errorf("appdata: writing %s: %w", path, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("appdata: %w", err)
	}

	return nil
}

// LegacyAudio is the sound preview volume as older versions kept it, in the
// settings file's [audio] table, before it moved to the database.
type LegacyAudio struct {
	Volume float64 `toml:"volume"`
	Muted  bool    `toml:"muted"`
}

// The bool reports whether the file has a valid [audio] table to move to the
// database. Settings no longer has it, so the next save drops it from the
// file.
func ReadLegacyAudio(path string) (LegacyAudio, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LegacyAudio{}, false, fmt.Errorf("appdata: %w", err)
	}

	var file struct {
		Audio *LegacyAudio `toml:"audio"`
	}

	if err := toml.Unmarshal(data, &file); err != nil {
		return LegacyAudio{}, false, fmt.Errorf("appdata: reading %s: %w", path, err)
	}

	if file.Audio == nil || file.Audio.Volume < 0 || file.Audio.Volume > 1 {
		return LegacyAudio{}, false, nil
	}

	return *file.Audio, true, nil
}
