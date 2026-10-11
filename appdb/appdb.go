// Package appdb is lockdeader's database: app state such as the sound preview
// volume, as opposed to the user's settings (see appdata), and the mods the
// user has installed.
//
// It's SQLite, through modernc.org/sqlite, which is pure Go. The schema is
// versioned (see migrations), so it can change between releases without
// losing what's stored.
package appdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// Open returns ErrNewerSchema for a database that a newer version of
// Lockdeader has upgraded. The database is left alone, so going back to the
// newer version loses nothing.
var ErrNewerSchema = errors.New("appdb: the database is from a newer version of Lockdeader")

// Its methods are safe to call from any goroutine.
type DB struct {
	db *sql.DB
}

// Every connection gets these, through the driver's _pragma parameters:
//   - foreign keys are enforced, so removing a mod removes its files;
//   - a write waits up to 5 seconds for another to finish, instead of failing;
//   - WAL journalling, with synchronous=NORMAL: commits are atomic and
//     survive crashes of the app; a power cut can lose only the last few.
const pragmas = "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"

// Open creates the database if it doesn't exist, and brings its schema up to
// date.
func Open(path string) (*DB, error) {
	// The driver takes its parameters after a "?", so the path can't have one.
	if strings.Contains(path, "?") {
		return nil, fmt.Errorf("appdb: can't open %s: the path contains a question mark", path)
	}

	return open(path + "?" + pragmas)
}

// OpenMemory is for carrying on when the real database can't be opened, and
// for tests.
func OpenMemory() (*DB, error) {
	return open(":memory:?" + pragmas)
}

func open(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("appdb: %w", err)
	}

	// One connection: writes never wait on each other within the app, and an
	// in-memory database (one per connection) stays one database.
	db.SetMaxOpenConns(1)

	d := &DB{db: db}

	if err := d.migrate(context.Background()); err != nil {
		db.Close()

		return nil, err
	}

	return d, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

// The schema's version is the number of migrations applied to it.
func (d *DB) Version() (int, error) {
	return version(context.Background(), d.db)
}

// The schema version is kept in SQLite's user_version, in the file's header.
func version(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (int, error) {
	var v int

	if err := q.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("appdb: reading the schema version: %w", err)
	}

	return v, nil
}

// Each migration runs in a transaction of its own along with the version it
// brings the schema to, so a crash can't leave a half-upgraded schema.
func (d *DB) migrate(ctx context.Context) error {
	current, err := version(ctx, d.db)
	if err != nil {
		return err
	}

	if current > len(migrations) {
		return fmt.Errorf("%w (its schema is version %d; this version knows up to %d)",
			ErrNewerSchema, current, len(migrations))
	}

	for v := current; v < len(migrations); v++ {
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("appdb: %w", err)
		}

		if err := migrations[v](ctx, tx); err != nil {
			tx.Rollback()

			return fmt.Errorf("appdb: upgrading the schema to version %d: %w", v+1, err)
		}

		// PRAGMA takes no parameters; v+1 is our own number.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", v+1)); err != nil {
			tx.Rollback()

			return fmt.Errorf("appdb: upgrading the schema to version %d: %w", v+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("appdb: upgrading the schema to version %d: %w", v+1, err)
		}
	}

	return nil
}
