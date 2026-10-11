package appdb

import (
	"context"
	"database/sql"
)

// A migration brings the schema up one version, inside the transaction it's
// given.
type migration func(ctx context.Context, tx *sql.Tx) error

// migrations[i] brings the schema from version i to version i+1, so the
// latest version is len(migrations). To change the schema, append a
// migration; never change one that's been released, as databases out there
// have already had it. A migration can be plain SQL (execSQL), or Go code
// for changes SQL can't express, e.g. moving data into a new shape.
var migrations = []migration{
	execSQL(schemaV1),
	execSQL(schemaV2),
}

func execSQL(statements string) migration {
	return func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, statements)

		return err
	}
}

// Every table is STRICT, so SQLite rejects values of the wrong type instead
// of storing them anyway. Times are Unix seconds.
const schemaV1 = `
-- App state that isn't a setting, by key: e.g. the sound preview volume.
CREATE TABLE app_state (
    key   TEXT PRIMARY KEY,
    value ANY
) STRICT;

-- The mods installed from GameBanana, one row each. A mod is here once its
-- files are in the cache (Cache/Mods); enabled means the game loads it.
CREATE TABLE mods (
    id            INTEGER PRIMARY KEY,
    section       TEXT    NOT NULL,            -- GameBanana section: 'Mod' or 'Sound'
    gamebanana_id INTEGER NOT NULL,            -- its id in that section
    name          TEXT    NOT NULL,
    author        TEXT    NOT NULL DEFAULT '',
    enabled       INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    load_order    INTEGER NOT NULL DEFAULT 0,  -- among enabled mods, lower wins
    installed_at  INTEGER NOT NULL,
    UNIQUE (section, gamebanana_id)
) STRICT;

-- The archives downloaded from GameBanana for each mod: enough to download
-- one again, and to check that the download is intact.
CREATE TABLE mod_files (
    id            INTEGER PRIMARY KEY,
    mod_id        INTEGER NOT NULL REFERENCES mods (id) ON DELETE CASCADE,
    gamebanana_id INTEGER NOT NULL UNIQUE,     -- the file's id: gamebanana.com/dl/<id>
    name          TEXT    NOT NULL,
    size          INTEGER NOT NULL,            -- in bytes
    md5           TEXT    NOT NULL,            -- GameBanana's checksum of the archive, in hex
    uploaded_at   INTEGER NOT NULL,
    downloaded_at INTEGER NOT NULL
) STRICT;

CREATE INDEX mod_files_mod_id ON mod_files (mod_id);

-- The VPKs taken out of each archive. Each is kept in the cache, named by
-- its SHA-256, so the cached copy can be checked against it.
CREATE TABLE mod_vpks (
    id          INTEGER PRIMARY KEY,
    file_id     INTEGER NOT NULL REFERENCES mod_files (id) ON DELETE CASCADE,
    path        TEXT    NOT NULL,              -- where it is in the archive
    sha256      TEXT    NOT NULL,              -- in hex; also its name in the cache
    size        INTEGER NOT NULL,              -- in bytes
    enabled     INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),  -- which of an archive's variants are used
    verified_at INTEGER,                       -- when the cached copy last matched sha256; NULL if never checked
    UNIQUE (file_id, path)
) STRICT;

CREATE INDEX mod_vpks_sha256 ON mod_vpks (sha256);
`

// What the installed mods page shows of each mod, besides its name and
// author, so it can show them without asking GameBanana.
const schemaV2 = `
ALTER TABLE mods ADD COLUMN thumbnail  TEXT    NOT NULL DEFAULT '';  -- URL of its preview image; '' if it has none
ALTER TABLE mods ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;   -- when the installed version was last updated on GameBanana
ALTER TABLE mods ADD COLUMN nsfw       INTEGER NOT NULL DEFAULT 0 CHECK (nsfw IN (0, 1));  -- GameBanana marks it not safe for work
`
