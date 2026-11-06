package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// ErrDatabaseNewer means the database was migrated by a newer version of the
// app. Opening it anyway could corrupt data this version does not understand.
var ErrDatabaseNewer = errors.New("storage: database was created by a newer version of the app")

var migrationName = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.sql$`)

type migration struct {
	version int
	name    string
	sql     string
}

// loadMigrations reads NNNN_name.sql files from dir. Versions must start at 1
// and have no gaps, so a missing file is caught before it reaches a user's data.
func loadMigrations(fsys fs.FS, dir string) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("storage: read migrations: %w", err)
	}

	var migrations []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrationName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("storage: migration file %q does not match NNNN_name.sql", e.Name())
		}
		version, _ := strconv.Atoi(m[1])
		body, err := fs.ReadFile(fsys, dir+"/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("storage: read migration %s: %w", e.Name(), err)
		}
		migrations = append(migrations, migration{version: version, name: m[2], sql: string(body)})
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	for i, m := range migrations {
		if want := i + 1; m.version != want {
			return nil, fmt.Errorf("storage: migration versions must be 1..N without gaps or duplicates, found %04d where %04d was expected", m.version, want)
		}
	}
	return migrations, nil
}

// migrate applies every migration that is not recorded yet. Each one runs in
// its own transaction together with its bookkeeping row, so a failure leaves
// the database at the previous version.
func migrate(ctx context.Context, db *sql.DB, migrations []migration) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("storage: create schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("storage: read schema version: %w", err)
	}
	if current > len(migrations) {
		return fmt.Errorf("%w (database version %d, this app knows %d)", ErrDatabaseNewer, current, len(migrations))
	}

	for _, m := range migrations[current:] {
		if err := apply(ctx, db, m); err != nil {
			return fmt.Errorf("storage: migration %04d_%s: %w", m.version, m.name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.version, m.name, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	return tx.Commit()
}
