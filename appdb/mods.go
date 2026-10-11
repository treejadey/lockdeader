package appdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Mod struct {
	ID           int64  // set by AddMod
	Section      string // GameBanana section: "Mod" or "Sound"
	GameBananaID int    // its id in that section
	Name         string
	Author       string
	Enabled      bool // the game loads it
	LoadOrder    int  // among enabled mods, lower wins
	InstalledAt  time.Time
	Thumbnail    string    // URL of its preview image; "" if it has none
	UpdatedAt    time.Time // when the installed version was last updated on GameBanana
	NSFW         bool      // GameBanana marks it not safe for work
	Files        []ModFile
}

// A ModFile is an archive downloaded from GameBanana: enough to download it
// again, and to check that the download is intact.
type ModFile struct {
	ID           int64 // set by AddMod or AddModFiles
	GameBananaID int   // the file's id: gamebanana.com/dl/<id>
	Name         string
	Size         int64  // in bytes
	MD5          string // GameBanana's checksum of the archive, in hex
	UploadedAt   time.Time
	DownloadedAt time.Time
	VPKs         []VPK
}

// A VPK is kept in the cache, named by its SHA-256.
type VPK struct {
	ID         int64     // set by AddMod or AddModFiles
	Path       string    // where it is in the archive
	SHA256     string    // in hex
	Size       int64     // in bytes
	Enabled    bool      // for archives with variants: whether this one is used
	VerifiedAt time.Time // when the cached copy last matched SHA256; zero if never checked
}

// AddMod returns m with its IDs set. It fails if the mod, or one of its files,
// is already stored.
func (d *DB) AddMod(ctx context.Context, m Mod) (Mod, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return Mod{}, fmt.Errorf("appdb: %w", err)
	}

	defer tx.Rollback() // no-op once committed

	res, err := tx.ExecContext(ctx, `
		INSERT INTO mods (section, gamebanana_id, name, author, enabled, load_order, installed_at,
		                  thumbnail, updated_at, nsfw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Section, m.GameBananaID, m.Name, m.Author, m.Enabled, m.LoadOrder, m.InstalledAt.Unix(),
		m.Thumbnail, m.UpdatedAt.Unix(), m.NSFW)
	if err != nil {
		return Mod{}, fmt.Errorf("appdb: adding mod %s/%d: %w", m.Section, m.GameBananaID, err)
	}

	if m.ID, err = res.LastInsertId(); err != nil {
		return Mod{}, fmt.Errorf("appdb: %w", err)
	}

	if m.Files, err = addFiles(ctx, tx, m.ID, m.Files); err != nil {
		return Mod{}, err
	}

	if err := tx.Commit(); err != nil {
		return Mod{}, fmt.Errorf("appdb: %w", err)
	}

	return m, nil
}

// AddModFiles is for files newly downloaded for a mod that's already
// installed. It returns them with their IDs set, and fails if one of them is
// already stored.
func (d *DB) AddModFiles(ctx context.Context, modID int64, files []ModFile) ([]ModFile, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("appdb: %w", err)
	}

	defer tx.Rollback() // no-op once committed

	if files, err = addFiles(ctx, tx, modID, files); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("appdb: %w", err)
	}

	return files, nil
}

// addFiles returns copies of files with their IDs set.
func addFiles(ctx context.Context, tx *sql.Tx, modID int64, files []ModFile) ([]ModFile, error) {
	files = append([]ModFile(nil), files...)

	for i := range files {
		f := &files[i]

		res, err := tx.ExecContext(ctx, `
			INSERT INTO mod_files (mod_id, gamebanana_id, name, size, md5, uploaded_at, downloaded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			modID, f.GameBananaID, f.Name, f.Size, f.MD5, f.UploadedAt.Unix(), f.DownloadedAt.Unix())
		if err != nil {
			return nil, fmt.Errorf("appdb: adding file %d: %w", f.GameBananaID, err)
		}

		if f.ID, err = res.LastInsertId(); err != nil {
			return nil, fmt.Errorf("appdb: %w", err)
		}

		f.VPKs = append([]VPK(nil), f.VPKs...)

		for j := range f.VPKs {
			v := &f.VPKs[j]

			res, err := tx.ExecContext(ctx, `
				INSERT INTO mod_vpks (file_id, path, sha256, size, enabled, verified_at)
				VALUES (?, ?, ?, ?, ?, ?)`,
				f.ID, v.Path, v.SHA256, v.Size, v.Enabled, nullTime(v.VerifiedAt))
			if err != nil {
				return nil, fmt.Errorf("appdb: adding %s from file %d: %w", v.Path, f.GameBananaID, err)
			}

			if v.ID, err = res.LastInsertId(); err != nil {
				return nil, fmt.Errorf("appdb: %w", err)
			}
		}
	}

	return files, nil
}

// The mods come in load order.
func (d *DB) Mods(ctx context.Context) ([]Mod, error) {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("appdb: %w", err)
	}

	defer tx.Rollback()

	var mods []Mod
	byID := make(map[int64]int) // index in mods

	err = query(ctx, tx, `
		SELECT id, section, gamebanana_id, name, author, enabled, load_order, installed_at,
		       thumbnail, updated_at, nsfw
		FROM mods ORDER BY load_order, id`,
		func(rows *sql.Rows) error {
			var m Mod
			var installed, updated int64

			if err := rows.Scan(&m.ID, &m.Section, &m.GameBananaID, &m.Name, &m.Author,
				&m.Enabled, &m.LoadOrder, &installed, &m.Thumbnail, &updated, &m.NSFW); err != nil {
				return err
			}

			m.InstalledAt, m.UpdatedAt = time.Unix(installed, 0), time.Unix(updated, 0)

			byID[m.ID] = len(mods)
			mods = append(mods, m)

			return nil
		})
	if err != nil {
		return nil, err
	}

	type fileRef struct{ mod, file int } // indexes in mods and mods[mod].Files
	files := make(map[int64]fileRef)

	err = query(ctx, tx, `
		SELECT id, mod_id, gamebanana_id, name, size, md5, uploaded_at, downloaded_at
		FROM mod_files ORDER BY id`,
		func(rows *sql.Rows) error {
			var f ModFile
			var modID, uploaded, downloaded int64

			if err := rows.Scan(&f.ID, &modID, &f.GameBananaID, &f.Name, &f.Size, &f.MD5,
				&uploaded, &downloaded); err != nil {
				return err
			}

			f.UploadedAt, f.DownloadedAt = time.Unix(uploaded, 0), time.Unix(downloaded, 0)

			m := byID[modID]
			files[f.ID] = fileRef{m, len(mods[m].Files)}
			mods[m].Files = append(mods[m].Files, f)

			return nil
		})
	if err != nil {
		return nil, err
	}

	err = query(ctx, tx, `
		SELECT id, file_id, path, sha256, size, enabled, verified_at
		FROM mod_vpks ORDER BY id`,
		func(rows *sql.Rows) error {
			var v VPK
			var fileID int64
			var verified sql.NullInt64

			if err := rows.Scan(&v.ID, &fileID, &v.Path, &v.SHA256, &v.Size, &v.Enabled, &verified); err != nil {
				return err
			}

			if verified.Valid {
				v.VerifiedAt = time.Unix(verified.Int64, 0)
			}

			ref := files[fileID]
			f := &mods[ref.mod].Files[ref.file]
			f.VPKs = append(f.VPKs, v)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return mods, nil
}

func (d *DB) SetModEnabled(ctx context.Context, id int64, enabled bool) error {
	return d.update(ctx, "UPDATE mods SET enabled = ? WHERE id = ?", enabled, id)
}

// ConfigureMod removes the files with the IDs in remove, along with their
// VPKs, and of the VPKs left, uses those with the IDs in enable, and none of
// the others. At least one VPK must be enabled, and every ID must be the
// mod's. It's all or nothing: on failure, the mod is left as it was. Deleting
// removed VPKs from the cache is up to the caller.
func (d *DB) ConfigureMod(ctx context.Context, modID int64, remove, enable []int64) error {
	if len(enable) == 0 {
		return fmt.Errorf("appdb: mod %d would have no VPKs enabled", modID)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	defer tx.Rollback() // no-op once committed

	for _, id := range remove {
		if err := updateOne(ctx, tx, "DELETE FROM mod_files WHERE id = ? AND mod_id = ?", id, modID); err != nil {
			return fmt.Errorf("appdb: removing file %d of mod %d: %w", id, modID, err)
		}
	}

	// Turn off every VPK the mod has left, then turn on the ones picked.
	_, err = tx.ExecContext(ctx, `
		UPDATE mod_vpks SET enabled = 0
		WHERE file_id IN (SELECT id FROM mod_files WHERE mod_id = ?)`, modID)
	if err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	for _, id := range enable {
		if err := updateOne(ctx, tx, `
			UPDATE mod_vpks SET enabled = 1
			WHERE id = ? AND file_id IN (SELECT id FROM mod_files WHERE mod_id = ?)`, id, modID); err != nil {
			return fmt.Errorf("appdb: VPK %d of mod %d: %w", id, modID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	return nil
}

func (d *DB) SetVPKVerified(ctx context.Context, id int64, at time.Time) error {
	return d.update(ctx, "UPDATE mod_vpks SET verified_at = ? WHERE id = ?", at.Unix(), id)
}

// The mod's files and VPKs go with it, but deleting them from the cache is up
// to the caller.
func (d *DB) RemoveMod(ctx context.Context, id int64) error {
	return d.update(ctx, "DELETE FROM mods WHERE id = ?", id)
}

// If no installed mod has a VPK with this SHA-256, its copy in the cache is no
// longer needed.
func (d *DB) VPKInUse(ctx context.Context, sha256 string) (bool, error) {
	var inUse bool

	err := d.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM mod_vpks WHERE sha256 = ?)", sha256).Scan(&inUse)
	if err != nil {
		return false, fmt.Errorf("appdb: %w", err)
	}

	return inUse, nil
}

// The statement should change exactly one row, whose id is the last argument.
func (d *DB) update(ctx context.Context, statement string, args ...any) error {
	err := updateOne(ctx, d.db, statement, args...)
	if errors.Is(err, errNoRow) {
		return fmt.Errorf("appdb: no row with id %v", args[len(args)-1])
	}

	if err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	return nil
}

var errNoRow = errors.New("no such row")

func updateOne(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, statement string, args ...any) error {
	res, err := db.ExecContext(ctx, statement, args...)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return errNoRow
	}

	return nil
}

func query(ctx context.Context, tx *sql.Tx, q string, row func(*sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, q)
	if err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		if err := row(rows); err != nil {
			return fmt.Errorf("appdb: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("appdb: %w", err)
	}

	return nil
}

func nullTime(t time.Time) sql.NullInt64 {
	if t.IsZero() {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}
