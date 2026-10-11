package deploy

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The game loads mods only if gameinfo.gi puts their folder on the search
// path, before "Game citadel". Lockdeader's lines there end in this mark,
// so they can be told from anyone else's and removed exactly. Only lines
// with the mark are ever touched.
//
// The first Game path is also where the game reads its cfg files from and
// writes its shader cache and reports to, unless Mod and Write paths say
// otherwise. With ours first, it would look for cfg/user_keys_default.vcfg
// in our folder and refuse to start, so Mod and Write lines for citadel go
// in too, as Deadlock Mod Manager's block has.
const (
	ourMark    = "// Lockdeader"
	ourLine    = "Game " + SearchPath + " " + ourMark
	addonsPath = "citadel/addons"
	ourAddon   = "Game " + addonsPath + " " + ourMark
	ourMod     = "Mod citadel " + ourMark
	ourWrite   = "Write citadel " + ourMark
	// Deadlock Mod Manager's block is between lines with this in them.
	dmmMarker = "// Deadlock Mod Manager"
)

// State is what gameinfo.gi says about mods.
type State struct {
	Enabled bool // the game loads our folder: there's a Game citadel/lockdeader line before Game citadel
	Addons  bool // the game loads citadel/addons too, whoever put the line in
	DMM     bool // Deadlock Mod Manager has written to the file
}

func GameInfoPath(gameDir string) string {
	return filepath.Join(gameDir, "game", "citadel", "gameinfo.gi")
}

// A gameInfo is gameinfo.gi split into lines, each with its own line
// ending, with the SearchPaths block found in them.
type gameInfo struct {
	lines  []string
	open   int // the line with the block's {
	close  int // the line with its }
	anchor int // the first "Game citadel" line in it: our lines go above
}

func parseGameInfo(data []byte) (*gameInfo, error) {
	lines := strings.SplitAfter(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	gi := &gameInfo{lines: lines, open: -1, close: -1, anchor: -1}
	depth := 0
	started := false

	for i, line := range lines {
		t := tokens(line)

		if !started {
			if len(t) == 0 || !strings.EqualFold(t[0], "SearchPaths") {
				continue
			}

			// The brace may be on the same line, or one of its own.
			started = true
			t = t[1:]
		}

		for _, tok := range t {
			switch tok {
			case "{":
				if depth == 0 {
					gi.open = i
				}

				depth++
			case "}":
				depth--

				if depth == 0 && gi.open >= 0 {
					gi.close = i
				}
			}
		}

		if gi.close >= 0 {
			break
		}

		if gi.open >= 0 && i > gi.open && gi.anchor < 0 && isGameLine(t, "citadel") {
			gi.anchor = i
		}
	}

	if gi.open < 0 || gi.close < 0 || gi.anchor < 0 || gi.anchor > gi.close {
		return nil, ErrNoSearchPaths
	}

	return gi, nil
}

// tokens splits a KeyValues line into its words, without any comment or
// quotes.
func tokens(line string) []string {
	line, _, _ = strings.Cut(line, "//")
	fields := strings.Fields(line)

	for i, f := range fields {
		fields[i] = strings.Trim(f, `"`)
	}

	return fields
}

func isGameLine(t []string, path string) bool {
	return isPathLine(t, "Game", path)
}

func isPathLine(t []string, key, path string) bool {
	return len(t) == 2 && strings.EqualFold(t[0], key) && strings.EqualFold(t[1], path)
}

func isOurs(line string) bool {
	collapsed := strings.Join(strings.Fields(line), " ")

	return collapsed == ourLine || collapsed == ourAddon || collapsed == ourMod || collapsed == ourWrite
}

func (gi *gameInfo) block() []string {
	return gi.lines[gi.open+1 : gi.close]
}

// has tells whether the block has a line with the key and path, ours or not.
func (gi *gameInfo) has(key, path string) bool {
	return slices.ContainsFunc(gi.block(), func(line string) bool {
		return isPathLine(tokens(line), key, path)
	})
}

func withoutOurs(lines []string) []string {
	return slices.DeleteFunc(slices.Clone(lines), isOurs)
}

func join(lines []string) []byte {
	return []byte(strings.Join(lines, ""))
}

// ourLineLike makes one of our lines in the style of the anchor line: its
// indentation, the gap between its words and its line ending.
func ourLineLike(anchor string, lines []string, key, path string) string {
	trimmed := strings.TrimLeft(anchor, " \t")
	lead := anchor[:len(anchor)-len(trimmed)]
	gap := "\t\t\t\t"

	if rest, ok := strings.CutPrefix(trimmed, "Game"); ok {
		if n := len(rest) - len(strings.TrimLeft(rest, " \t")); n > 0 {
			gap = rest[:n]
		}
	}

	ending := lineEnding(anchor)

	// The anchor can be the file's last line, without a line ending of its
	// own; ours still needs one.
	if ending == "" {
		ending = "\n"

		if slices.ContainsFunc(lines, func(l string) bool { return strings.HasSuffix(l, "\r\n") }) {
			ending = "\r\n"
		}
	}

	return lead + key + gap + path + "\t" + ourMark + ending
}

func lineEnding(line string) string {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(line, "\n"):
		return "\n"
	}

	return ""
}

func ReadGameInfo(gameDir string) (State, error) {
	data, err := os.ReadFile(GameInfoPath(gameDir))
	if err != nil {
		return State{}, fmt.Errorf("deploy: %w", err)
	}

	gi, err := parseGameInfo(data)
	if err != nil {
		return State{}, err
	}

	s := State{
		Addons: gi.has("Game", addonsPath),
		DMM:    bytes.Contains(data, []byte(dmmMarker)),
	}

	// Only a line before Game citadel makes the game load our folder.
	for _, line := range gi.lines[gi.open+1 : gi.anchor] {
		if isGameLine(tokens(line), SearchPath) {
			s.Enabled = true
		}
	}

	return s, nil
}

// EnableMods puts our lines into gameinfo.gi: one for our folder, and one
// for citadel/addons too if addons is set and there isn't one already, so
// mods installed outside Lockdeader keep loading. It goes after ours, so
// ours win. Mod and Write lines for citadel go in as well, unless the file
// has them (see ourMark). Before the file is first written, it's copied to
// backup, unless backup is "". changed is false if the lines were there
// already.
func EnableMods(gameDir string, addons bool, backup string) (changed bool, err error) {
	path := GameInfoPath(gameDir)

	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("deploy: %w", err)
	}

	gi, err := parseGameInfo(data)
	if err != nil {
		return false, err
	}

	// Start from the file without our lines, in case they've moved.
	kept, err := parseGameInfo(join(withoutOurs(gi.lines)))
	if err != nil {
		return false, err
	}

	anchor := kept.lines[kept.anchor]
	want := []string{ourLineLike(anchor, kept.lines, "Game", SearchPath)}

	if addons && !kept.has("Game", addonsPath) {
		want = append(want, ourLineLike(anchor, kept.lines, "Game", addonsPath))
	}

	for _, key := range []string{"Mod", "Write"} {
		if !kept.has(key, "citadel") {
			want = append(want, ourLineLike(anchor, kept.lines, key, "citadel"))
		}
	}

	current := slices.DeleteFunc(slices.Clone(gi.lines), func(line string) bool { return !isOurs(line) })
	if slices.Equal(current, want) {
		return false, nil
	}

	out := slices.Concat(kept.lines[:kept.anchor], want, kept.lines[kept.anchor:])

	if err := validate(out, kept.lines, 1); err != nil {
		return false, err
	}

	if err := writeGameInfo(path, data, join(out), backup); err != nil {
		return false, err
	}

	return true, nil
}

// DisableMods takes our lines out of gameinfo.gi, leaving the file as it was
// before them.
func DisableMods(gameDir string, backup string) (changed bool, err error) {
	path := GameInfoPath(gameDir)

	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("deploy: %w", err)
	}

	lines := strings.SplitAfter(string(data), "\n")
	out := withoutOurs(lines)

	if len(out) == len(lines) {
		return false, nil
	}

	if err := validate(out, out, 0); err != nil {
		return false, err
	}

	if err := writeGameInfo(path, data, join(out), backup); err != nil {
		return false, err
	}

	return true, nil
}

// validate checks an edit before it's written: the file still has its
// SearchPaths block, it has exactly ours of our folder's lines, and nothing
// but our lines differs from base. gameinfo.gi is the one file a mistake
// in would stop the game starting.
func validate(out, base []string, ours int) error {
	if _, err := parseGameInfo(join(out)); err != nil {
		return err
	}

	count := 0

	for _, line := range out {
		if strings.Join(strings.Fields(line), " ") == ourLine {
			count++
		}
	}

	if count != ours {
		return fmt.Errorf("deploy: gameinfo.gi would have %d of our lines, not %d", count, ours)
	}

	if !bytes.Equal(join(withoutOurs(out)), join(withoutOurs(base))) {
		return errors.New("deploy: the edit to gameinfo.gi would change more than our lines")
	}

	return nil
}

func writeGameInfo(path string, before, after []byte, backup string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	if backup != "" {
		if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return fmt.Errorf("deploy: %w", err)
			}

			if err := writeFileAtomic(backup, before, 0o644); err != nil {
				return err
			}
		}
	}

	return writeFileAtomic(path, after, info.Mode().Perm())
}
