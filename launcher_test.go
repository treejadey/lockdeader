package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/game"
)

func TestGameLauncher(t *testing.T) {
	mainthread.Wait(func() {
		var launched [][]string
		var opened []string
		launchErr := error(nil)

		l := newGameLauncher(func() string { return `-novid +exec "my config.cfg"` })
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

		if got := l.Value("message").ToString(); got != "Starting Deadlock" {
			t.Errorf("message = %q", got)
		}

		// Without Steam's executable, through a steam:// link.
		launchErr = game.ErrNoSteam
		l.Play(false)

		if len(opened) != 1 || opened[0] != "steam://run/1422450//-novid%20+exec%20my%20config.cfg/" {
			t.Errorf("opened %q, want a steam:// link with the launch options", opened)
		}

		if got := l.Value("messageId").ToInt(); got != 2 {
			t.Errorf("messageId = %d, want 2", got)
		}

		launchErr = errors.New("exec: permission denied")
		l.Play(false)

		if got := l.Value("message").ToString(); got == "Starting Deadlock" {
			t.Error("a failed start says it's starting")
		}
	})
}
