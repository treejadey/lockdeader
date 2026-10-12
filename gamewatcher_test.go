package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mappu/miqt/qt6/mainthread"
)

func TestGameWatcher(t *testing.T) {
	// Touched on the main thread, other than running, which the checks read
	// while the test waits.
	results := make(chan error, 1)
	running := false
	var changes []bool
	var w *gameWatcher

	mainthread.Wait(func() {
		w = newGameWatcher(func(r bool) { changes = append(changes, r) })
		w.running = func() (bool, error) {
			err := <-results

			return running, err
		}
	})

	check := func(r bool, err error) {
		t.Helper()

		running = r
		mainthread.Wait(w.Check)
		results <- err
		waitFor(t, "the check", func() bool { return !w.checking })
	}

	// Only changes are reported.
	check(false, nil)
	check(true, nil)
	check(true, nil)

	// A failed check isn't taken for the game closing.
	check(false, errors.New("permission denied"))
	check(false, nil)

	mainthread.Wait(func() {
		if fmt.Sprint(changes) != "[true false]" {
			t.Errorf("changes = %v, want [true false]", changes)
		}
	})
}
