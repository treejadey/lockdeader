package main

import (
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/deploy"
)

// gameWatcher checks every two seconds whether Deadlock is running, and
// calls onChange when that changes.
//
// All methods must be called on the Qt main thread.
type gameWatcher struct {
	onChange func(running bool)

	// deploy.Running, other than in tests.
	running func() (bool, error)

	timer    *qt.QTimer
	checking bool
	current  bool
	failed   bool
}

func newGameWatcher(onChange func(running bool)) *gameWatcher {
	w := &gameWatcher{
		onChange: onChange,
		running:  deploy.Running,
		timer:    qt.NewQTimer(),
	}

	w.timer.SetInterval(2000)
	w.timer.OnTimeout(w.Check)

	return w
}

// Start checks now, then every two seconds.
func (w *gameWatcher) Start() {
	w.timer.Start2()
	w.Check()
}

// Check looks off the main thread, since it goes through every process. A
// check that's still going isn't started again.
func (w *gameWatcher) Check() {
	if w.checking {
		return
	}

	w.checking = true

	go func() {
		running, err := w.running()

		mainthread.Start(func() { w.checked(running, err) })
	}()
}

func (w *gameWatcher) checked(running bool, err error) {
	w.checking = false

	// A check that fails changes nothing; above all, it isn't taken for the
	// game closing. It's logged once, not every two seconds.
	if err != nil {
		if !w.failed {
			slog.Warn("couldn't tell whether Deadlock is running", "err", err)
		}

		w.failed = true

		return
	}

	w.failed = false

	if running == w.current {
		return
	}

	w.current = running
	w.onChange(running)
}
