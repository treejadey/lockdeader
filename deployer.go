package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
)

// errGameFolder wraps an error from changing the game folder, to tell it
// from one in the data folder, e.g. for permission errors.
var errGameFolder = errors.New("in the game folder")

// deployer puts the mods that are turned on into the game (deploy.Apply):
// half a second after each change, like the settings are saved, and before
// Deadlock is started with mods. For Lockdeader's own launches it also puts
// its lines into gameinfo.gi, or takes them out (Prepare).
//
// All methods must be called on the Qt main thread.
type deployer struct {
	db            *appdb.DB
	store         install.Store
	gameDir       func() string // "" while there's no game folder
	loadOtherMods func() bool
	backup        string // where gameinfo.gi is copied before it's first written

	// The deploy package's functions, other than in tests.
	apply   func(gameDir string, files []deploy.File) (deploy.Result, error)
	running func() (bool, error)
	enable  func(gameDir string, addons bool, backup string) (bool, error)
	disable func(gameDir, backup string) (bool, error)

	// Called after each apply with how it went: nil, deploy.ErrGameRunning,
	// or what went wrong. May be nil.
	onDone func(err error)
	// Called after each apply that got as far as merging the tables mods
	// share, even while the game runs: for each mod, by key, that the merge
	// left some changes out of, the keys of the mods it's merged with. May
	// be nil.
	onMerged func(leftOut map[string][]string)

	timer *qt.QTimer
	busy  bool
	// The next apply takes Lockdeader's lines out of gameinfo.gi first.
	takeOutLines bool
	// A change came in while applying, so it goes again once done, and the
	// callbacks waiting for that run.
	again   bool
	current []func(error)
	pending []func(error)
}

func newDeployer(db *appdb.DB, store install.Store, gameDir func() string, loadOtherMods func() bool, backup string) *deployer {
	d := &deployer{
		db:            db,
		store:         store,
		gameDir:       gameDir,
		loadOtherMods: loadOtherMods,
		backup:        backup,
		apply:         deploy.Apply,
		running:       deploy.Running,
		enable:        deploy.EnableMods,
		disable:       deploy.DisableMods,
		timer:         qt.NewQTimer(),
	}

	d.timer.SetSingleShot(true)
	d.timer.SetInterval(500)
	d.timer.OnTimeout(func() { d.Apply(nil) })

	return d
}

// ApplySoon applies half a second after the last change.
func (d *deployer) ApplySoon() {
	d.timer.Start2()
}

// ResetSoon takes Lockdeader's lines out of gameinfo.gi, which are only for
// its own launches, and applies, half a second after the last change. It's
// for when the game isn't running Lockdeader's launch any more: on startup,
// and once it closes. The lines come out as part of the apply, so they're
// out before Prepare puts them back in for a launch that follows.
func (d *deployer) ResetSoon() {
	d.takeOutLines = true
	d.ApplySoon()
}

// Flush applies now if a change is waiting to be.
func (d *deployer) Flush() {
	if d.timer.IsActive() {
		d.timer.Stop()
		d.Apply(nil)
	}
}

// Idle is false while an apply is running or waiting to.
func (d *deployer) Idle() bool {
	return !d.busy && !d.timer.IsActive()
}

// Apply puts the mods that are turned on into the game, off the main
// thread, and calls then (if not nil) on it with how it went. While
// Deadlock is running, nothing is changed: then gets deploy.ErrGameRunning,
// and the change waits for the next apply. With no game folder there's
// nothing to do, and then gets nil.
func (d *deployer) Apply(then func(error)) {
	dir := d.gameDir()
	if dir == "" {
		// Without the game, nothing is merged.
		if d.onMerged != nil {
			d.onMerged(nil)
		}

		if then != nil {
			then(nil)
		}

		return
	}

	if d.busy {
		d.again = true

		if then != nil {
			d.pending = append(d.pending, then)
		}

		return
	}

	d.busy = true
	d.timer.Stop()

	if then != nil {
		d.current = append(d.current, then)
	}

	takeOutLines := d.takeOutLines
	d.takeOutLines = false

	go func() {
		leftOut, err := d.run(dir, takeOutLines)

		mainthread.Start(func() { d.finish(leftOut, err) })
	}()
}

// run applies, and returns what the merge of shared tables left out (see
// onMerged), or nil if it didn't get that far.
func (d *deployer) run(dir string, takeOutLines bool) (map[string][]string, error) {
	// Even while the game runs: it has already read gameinfo.gi. A failure
	// is only logged, and doesn't stop the mods going in.
	if takeOutLines {
		if _, err := d.disable(dir, d.backup); err != nil {
			slog.Error("failed to take Lockdeader's lines out of gameinfo.gi", "err", err)
		}
	}

	mods, keys, err := d.mods()
	if err != nil {
		return nil, err
	}

	// Merged even while the game runs, which only reads, so what's left out
	// is known then too.
	files, leftOut := d.withMergedTables(dir, mods, keys)

	if running, err := d.running(); err != nil {
		slog.Warn("couldn't tell whether Deadlock is running", "err", err)
	} else if running {
		return leftOut, deploy.ErrGameRunning
	}

	res, err := d.apply(dir, files)

	slog.Info("put the mods into the game", "placed", res.Placed, "removed", res.Removed,
		"bundled", res.Bundled, "missing", len(res.Missing), "err", err)

	if err != nil {
		return leftOut, fmt.Errorf("%w: %w", errGameFolder, err)
	}

	return leftOut, nil
}

// mods lists the mods that are on, in load order, with the VPKs of theirs
// the game should load, and their keys.
func (d *deployer) mods() ([]deploy.Mod, []string, error) {
	mods, err := d.db.Mods(context.Background())
	if err != nil {
		return nil, nil, err
	}

	var on []deploy.Mod
	var keys []string

	for _, m := range mods {
		if !m.Enabled {
			continue
		}

		var mod deploy.Mod

		for _, f := range m.Files {
			for _, v := range f.VPKs {
				if v.Enabled {
					mod.Files = append(mod.Files, deploy.File{SHA256: v.SHA256, Path: d.store.Path(v.SHA256), Size: v.Size})
				}
			}
		}

		on = append(on, mod)
		keys = append(keys, installedKey(banana.Section(m.Section), m.GameBananaID))
	}

	return on, keys, nil
}

// withMergedTables lists the VPKs the game should load, in load order:
// the mods', after one holding the sound event tables they share, merged
// (see deploy.MergeTables), which wins those. It's kept in the store like
// the mods' VPKs, by its SHA-256, so the same mods make the same file and
// an apply that changes nothing rewrites nothing. Tables that can't be
// merged are left to the load order, as before merging.
//
// It also returns, for each mod by key that the merge left some changes
// out of, the keys of the mods it's merged with.
func (d *deployer) withMergedTables(dir string, mods []deploy.Mod, keys []string) ([]deploy.File, map[string][]string) {
	var files []deploy.File

	for _, m := range mods {
		files = append(files, m.Files...)
	}

	merged, err := deploy.MergeTables(dir, mods)
	if err != nil {
		slog.Warn("not everything was merged that might have been", "err", err)
	}

	leftOut := map[string][]string{}

	if merged.VPK == nil {
		return files, leftOut
	}

	sha, size, err := d.store.Add(bytes.NewReader(merged.VPK))
	if err != nil {
		slog.Error("failed to keep the merged tables", "err", err)

		return files, leftOut
	}

	slog.Info("merged the tables mods share", "tables", len(merged.Tables))

	for i, others := range merged.LeftOut {
		for _, j := range others {
			leftOut[keys[i]] = append(leftOut[keys[i]], keys[j])
		}
	}

	return append([]deploy.File{{SHA256: sha, Path: d.store.Path(sha), Size: size}}, files...), leftOut
}

func (d *deployer) finish(leftOut map[string][]string, err error) {
	d.busy = false

	if leftOut != nil && d.onMerged != nil {
		d.onMerged(leftOut)
	}

	if errors.Is(err, deploy.ErrGameRunning) {
		slog.Info("Deadlock is running, so the mods will be put into the game later")
	} else if err != nil {
		slog.Error("failed to put the mods into the game", "err", err)
	}

	for _, then := range d.current {
		then(err)
	}

	d.current = nil

	if d.onDone != nil {
		d.onDone(err)
	}

	if d.again {
		d.again = false
		d.current, d.pending = d.pending, nil
		d.Apply(nil)
	}
}

// Prepare gets the game ready to start: with mods, it applies them and puts
// our lines into gameinfo.gi; without, it takes the lines out. then is
// called on the main thread with any error, in which case the game
// shouldn't be started.
func (d *deployer) Prepare(withMods bool, then func(error)) {
	dir := d.gameDir()

	if !withMods {
		go func() {
			_, err := d.disable(dir, d.backup)

			mainthread.Start(func() { then(d.gameInfoError(err)) })
		}()

		return
	}

	d.Apply(func(err error) {
		if err != nil {
			then(err)

			return
		}

		addons := d.loadOtherMods()

		go func() {
			_, err := d.enable(dir, addons, d.backup)

			mainthread.Start(func() { then(d.gameInfoError(err)) })
		}()
	})
}

func (d *deployer) gameInfoError(err error) error {
	if err == nil {
		return nil
	}

	slog.Error("failed to change gameinfo.gi", "err", err)

	return fmt.Errorf("%w: %w", errGameFolder, err)
}
