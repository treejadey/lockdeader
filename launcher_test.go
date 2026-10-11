package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/game"
)

func TestGameLauncher(t *testing.T) {
	mainthread.Wait(func() {
		var launched [][]string
		var opened []string
		var prepared []bool
		launchErr := error(nil)
		prepareErr := error(nil)
		gameFolder := "/games/Deadlock"

		l := newGameLauncher(func() string { return `-novid +exec "my config.cfg"` },
			func() string { return gameFolder },
			func(withMods bool, then func(error)) {
				prepared = append(prepared, withMods)
				then(prepareErr)
			})
		l.launch = func(args []string) error {
			launched = append(launched, args)

			return launchErr
		}
		l.openURL = func(url string) bool {
			opened = append(opened, url)

			return true
		}

		l.Play(true)

		if fmt.Sprint(launched) != "[[-novid +exec my config.cfg]]" || len(opened) != 0 {
			t.Errorf("launched %q, opened %q; want the launch options split, through Steam", launched, opened)
		}

		if fmt.Sprint(prepared) != "[true]" {
			t.Errorf("prepared %v, want [true]", prepared)
		}

		if got := l.Value("message").ToString(); got != "Starting Deadlock with mods" {
			t.Errorf("message = %q", got)
		}

		// Without Steam's executable, through a steam:// link.
		launchErr = game.ErrNoSteam
		l.Play(false)

		if len(opened) != 1 || opened[0] != "steam://run/1422450//-novid%20+exec%20my%20config.cfg/" {
			t.Errorf("opened %q, want a steam:// link with the launch options", opened)
		}

		if fmt.Sprint(prepared) != "[true false]" {
			t.Errorf("prepared %v, want [true false]", prepared)
		}

		if got := l.Value("messageId").ToInt(); got != 2 {
			t.Errorf("messageId = %d, want 2", got)
		}

		launchErr = errors.New("exec: permission denied")
		l.Play(false)

		if got := l.Value("message").ToString(); got == "Starting Deadlock without mods" {
			t.Error("a failed start says it's starting")
		}

		// The game isn't started if it can't be got ready.
		launchErr = nil
		prepareErr = fmt.Errorf("%w: %w", errGameFolder, deploy.ErrMissing)
		l.Play(true)

		if len(launched) != 3 {
			t.Errorf("launched %d times after a failed prepare, want 3", len(launched))
		}

		if got := l.Value("message").ToString(); got != "Some mods’ files are missing. Reinstall them." {
			t.Errorf("message = %q", got)
		}

		prepareErr = deploy.ErrGameRunning
		l.Play(true)

		if got := l.Value("message").ToString(); got != "Deadlock is running. Close it and try again." {
			t.Errorf("message = %q", got)
		}

		// Without a game folder, the message offers the settings.
		prepareErr = nil
		gameFolder = ""
		l.Play(true)

		if len(launched) != 3 || len(prepared) != 5 {
			t.Error("the game was readied or started without a folder")
		}

		if got, action := l.Value("message").ToString(), l.Value("messageAction").ToString(); got != "Choose Deadlock’s folder first." || action != "settings" {
			t.Errorf("message = %q, action = %q", got, action)
		}

		gameFolder = "/games/Deadlock"
		l.Play(false)

		if got := l.Value("messageAction").ToString(); got != "" {
			t.Errorf("messageAction = %q after a plain message", got)
		}
	})
}
