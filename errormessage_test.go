package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"testing"

	"github.com/treejadey/lockdeader/audio"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
)

func TestErrorMessage(t *testing.T) {
	unreachable := "GameBanana can’t be reached. Check your internet connection and try again."

	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"no DNS", &url.Error{Op: "Get", URL: "https://gamebanana.com", Err: &net.DNSError{Err: "no such host", Name: "gamebanana.com"}}, unreachable},
		{"refused", fmt.Errorf("banana: %w", &url.Error{Op: "Get", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}), unreachable},
		{"stalled", fmt.Errorf("%s: %w", "mod.zip", install.ErrStalled), unreachable},
		{"404", &banana.APIError{StatusCode: 404}, "It’s no longer on GameBanana."},
		{"download 404", fmt.Errorf("banana: downloading x: %w", &banana.APIError{StatusCode: 404}), "It’s no longer on GameBanana."},
		{"429", &banana.APIError{StatusCode: 429}, "GameBanana is busy. Try again in a minute."},
		{"503", &banana.APIError{StatusCode: 503}, "GameBanana isn’t working right now. Try again later."},
		{"400", &banana.APIError{StatusCode: 400, Code: "INPUT_ERRORS"}, "Something went wrong."},
		{"checksum", fmt.Errorf("%s: %w", "mod.zip", install.ErrChecksum), "The download was damaged. Try installing it again."},
		{"unknown format", install.ErrUnknownFormat, "The mod’s file isn’t an archive Lockdeader can open."},
		{"no VPKs", fmt.Errorf("%s: %w", "mod.zip", install.ErrNoVPKs), "The mod has no files the game can use."},
		{"not a VPK", fmt.Errorf("install: %s: %w", "a.vpk", install.ErrNotVPK), "The mod has no files the game can use."},
		{"chunked", fmt.Errorf("%w (%s)", install.ErrChunkedVPK, "pak01_000.vpk"), "The mod is split into parts, which isn’t supported yet."},
		{"permission", fmt.Errorf("install: %w", &fs.PathError{Op: "open", Path: "/x", Err: os.ErrPermission}), "Lockdeader isn’t allowed to save files in its data folder."},
		{"too large", fmt.Errorf("%s: %w", "mod.zip", install.ErrTooLarge), "The mod is too big to pack."},
		{"game running", deploy.ErrGameRunning, "Deadlock is running. Close it and try again."},
		{"too many", fmt.Errorf("%w: %w", errGameFolder, deploy.ErrTooMany), "Too many of the mods that are on change the same files. Turn some off."},
		{"missing", fmt.Errorf("%w: %w", errGameFolder, fmt.Errorf("%w: abc", deploy.ErrMissing)), "Some mods’ files are missing. Reinstall them."},
		{"gameinfo", fmt.Errorf("%w: %w", errGameFolder, deploy.ErrNoSearchPaths), "Deadlock’s gameinfo.gi isn’t one Lockdeader understands. Verify the game files in Steam."},
		{"game folder permission", fmt.Errorf("%w: %w", errGameFolder, fmt.Errorf("deploy: %w", &fs.PathError{Op: "open", Path: "/g", Err: os.ErrPermission})), "Lockdeader isn’t allowed to change files in Deadlock’s folder."},
		{"no output", &audio.Error{Op: "opening the output device", Kind: audio.ErrDevice, Reason: "no backend"}, "No sound output is available."},
		{"bad sound", &audio.Error{Op: "decoding", Kind: audio.ErrDecode, Reason: "invalid data"}, "This sound can’t be played."},
		{"other", errors.New("something odd"), "Something went wrong."},
	} {
		if got := errorMessage(tc.err); got != tc.want {
			t.Errorf("%s: errorMessage(%v) = %q, want %q", tc.name, tc.err, got, tc.want)
		}
	}
}
