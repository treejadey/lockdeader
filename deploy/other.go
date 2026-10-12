package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Mods installed by other tools, or by hand, live in game/citadel/addons
// (and addons2 … addons10, which Deadlock Mod Manager spills over into).
// Lockdeader never changes them unless asked to take them over.

// A Source is who put a mod in the addons folder.
type Source string

const (
	// Listed in Deadlock Mod Manager's ledger, .dmm.json.
	SourceDMM Source = "dmm"
	// A VPK in the folder that no ledger lists: put there by hand, or by a
	// tool that keeps no ledger.
	SourceAddons Source = "addons"
)

// An OtherMod is one in the game's addons folders.
type OtherMod struct {
	Key    string // unique among a scan's results
	Source Source
	// Deadlock Mod Manager's id for it: a GameBanana mod id ("634732"),
	// "snd-<id>" for a sound, "local-<uuid>" for a file it imported; "" for
	// a loose file.
	DMMKey string
	// Its GameBanana id, from DMMKey; 0 when there's none.
	GameBananaID int
	// "Sound" when DMMKey says so; "" when the id could be a mod's or a
	// sound's (older ledgers don't say).
	Section  string
	Enabled  bool // the game loads it: a file of its is named pakNN_dir.vpk
	Files    []OtherFile
	Modified time.Time // its newest file
}

type OtherFile struct {
	Path    string // in the game folder
	Name    string // the file's name before the tool renamed it, if known
	Size    int64
	Enabled bool
}

const ledgerName = ".dmm.json"

var (
	enabledName = regexp.MustCompile(`(?i)^pak\d{2}_dir\.vpk$`)
	soundKey    = regexp.MustCompile(`^snd-(\d+)$`)
)

// AddonsFolder is game/citadel/addons, where other tools put mods.
func AddonsFolder(gameDir string) string {
	return filepath.Join(gameDir, "game", "citadel", "addons")
}

// addonsFolders lists game/citadel/addons and its spill-over folders, in
// the order the game searches them.
func addonsFolders(gameDir string) []string {
	folders := []string{AddonsFolder(gameDir)}

	for i := 2; i <= 10; i++ {
		folders = append(folders, filepath.Join(gameDir, "game", "citadel", "addons"+strconv.Itoa(i)))
	}

	return folders
}

// ScanOtherMods lists the mods in the game's addons folders: those Deadlock
// Mod Manager's ledger lists, grouped as it groups them, then every other
// VPK, one mod each. Mods that are on come first, in the order the game
// loads them.
func ScanOtherMods(gameDir string) ([]OtherMod, error) {
	folders := addonsFolders(gameDir)

	// Every VPK, by folder, before the ledger claims its share.
	type vpkFile struct {
		path    string
		size    int64
		modTime time.Time
	}

	loose := make(map[string]map[string]vpkFile, len(folders)) // folder → name → file

	for _, folder := range folders {
		entries, err := os.ReadDir(folder)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("deploy: %w", err)
		}

		files := make(map[string]vpkFile)

		for _, e := range entries {
			if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(e.Name()), ".vpk") {
				continue
			}

			info, err := e.Info()
			if err != nil {
				continue
			}

			files[e.Name()] = vpkFile{path: filepath.Join(folder, e.Name()), size: info.Size(), modTime: info.ModTime()}
		}

		loose[folder] = files
	}

	var mods []OtherMod

	// take moves a file out of loose into a mod, looking in the folder the
	// ledger says first, then anywhere.
	take := func(m *OtherMod, folder, name, original string, enabled bool) {
		for _, f := range append([]string{folder}, folders...) {
			file, ok := loose[f][name]
			if !ok {
				continue
			}

			delete(loose[f], name)

			if original == "" {
				original = name
			}

			m.Files = append(m.Files, OtherFile{Path: file.path, Name: original, Size: file.size, Enabled: enabled})
			m.Enabled = m.Enabled || enabled
			m.Modified = maxTime(m.Modified, file.modTime)

			return
		}
	}

	ledger, err := readLedger(filepath.Join(folders[0], ledgerName))
	if err != nil {
		return nil, err
	}

	for _, key := range slices.Sorted(mapsKeys(ledger.Mods)) {
		entry := ledger.Mods[key]
		m := OtherMod{Key: "dmm/" + key, Source: SourceDMM, DMMKey: key}
		m.GameBananaID, m.Section = parseDMMKey(key)

		folder := folders[0]
		if entry.Shard >= 2 && entry.Shard <= 10 {
			folder = folders[entry.Shard-1]
		}

		// The original names go with the files that are on, then those off,
		// which DMM renames to <id>_<original>.
		original := func(i int, name string) string {
			if i < len(entry.OriginalVPKNames) {
				return entry.OriginalVPKNames[i]
			}

			return strings.TrimPrefix(name, key+"_")
		}

		for i, name := range entry.CurrentVPKs {
			take(&m, folder, name, original(i, name), enabledName.MatchString(name))
		}

		for i, name := range entry.DisabledVPKs {
			take(&m, folder, name, original(len(entry.CurrentVPKs)+i, name), false)
		}

		// A ledger entry whose files are gone isn't a mod any more.
		if len(m.Files) > 0 {
			mods = append(mods, m)
		}
	}

	for _, folder := range folders {
		for _, name := range slices.Sorted(mapsKeys(loose[folder])) {
			file := loose[folder][name]
			rel, _ := filepath.Rel(filepath.Join(gameDir, "game", "citadel"), file.path)

			mods = append(mods, OtherMod{
				Key:      "addons/" + filepath.ToSlash(rel),
				Source:   SourceAddons,
				Enabled:  enabledName.MatchString(name),
				Files:    []OtherFile{{Path: file.path, Name: name, Size: file.size, Enabled: enabledName.MatchString(name)}},
				Modified: file.modTime,
			})
		}
	}

	// The game loads lower pak numbers first, folder by folder; mods that
	// are off go last, as they are.
	slices.SortStableFunc(mods, func(a, b OtherMod) int {
		if a.Enabled != b.Enabled {
			if a.Enabled {
				return -1
			}

			return 1
		}

		return strings.Compare(firstEnabledPath(a), firstEnabledPath(b))
	})

	return mods, nil
}

func firstEnabledPath(m OtherMod) string {
	for _, f := range m.Files {
		if f.Enabled {
			// With slashes, so "addons/" sorts before "addons2/" on Windows
			// too: '/' comes before '2', but '\\' comes after it.
			return strings.ToLower(filepath.ToSlash(f.Path))
		}
	}

	return ""
}

func parseDMMKey(key string) (id int, section string) {
	if n, err := strconv.Atoi(key); err == nil && n > 0 {
		return n, ""
	}

	if m := soundKey.FindStringSubmatch(key); m != nil {
		n, _ := strconv.Atoi(m[1])

		return n, "Sound"
	}

	return 0, ""
}

// Deadlock Mod Manager's ledger. Only what's needed is decoded; the rest of
// each entry is kept as it is when the file is rewritten.
type ledger struct {
	raw  map[string]json.RawMessage
	Mods map[string]ledgerEntry
}

type ledgerEntry struct {
	Enabled          bool     `json:"enabled"`
	Shard            int      `json:"shard"`
	CurrentVPKs      []string `json:"currentVpks"`
	DisabledVPKs     []string `json:"disabledVpks"`
	OriginalVPKNames []string `json:"originalVpkNames"`
}

// readLedger returns an empty ledger for a missing file. A file that can't
// be understood is reported: its mods would otherwise look like loose files.
func readLedger(path string) (ledger, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ledger{}, nil
	}

	if err != nil {
		return ledger{}, fmt.Errorf("deploy: %w", err)
	}

	var l ledger

	if err := json.Unmarshal(data, &l.raw); err != nil {
		return ledger{}, fmt.Errorf("deploy: reading %s: %w", path, err)
	}

	if mods, ok := l.raw["mods"]; ok {
		if err := json.Unmarshal(mods, &l.Mods); err != nil {
			return ledger{}, fmt.Errorf("deploy: reading %s: %w", path, err)
		}
	}

	return l, nil
}

// ForgetInLedger takes mods out of Deadlock Mod Manager's ledger in the
// game's addons folder, once their files have been taken over, so it doesn't
// go looking for them. With no mods left, the ledger is deleted.
func ForgetInLedger(gameDir string, dmmKeys []string) error {
	path := filepath.Join(addonsFolders(gameDir)[0], ledgerName)

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("deploy: reading %s: %w", path, err)
	}

	var mods map[string]json.RawMessage

	if entries, ok := raw["mods"]; ok {
		if err := json.Unmarshal(entries, &mods); err != nil {
			return fmt.Errorf("deploy: reading %s: %w", path, err)
		}
	}

	for _, key := range dmmKeys {
		delete(mods, key)
	}

	if len(mods) == 0 {
		return removeFile(path)
	}

	if raw["mods"], err = json.Marshal(mods); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return writeFileAtomic(path, out, 0o644)
}

func mapsKeys[V any](m map[string]V) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}

	return a
}
