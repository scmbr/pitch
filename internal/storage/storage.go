// Package storage opens the SQLite database and keeps its schema up to date.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite" // pure Go driver, registers itself as "sqlite"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

const fileName = "pitch.sqlite"

// Path returns the database file inside the data folder.
func Path(dataDir string) string {
	return filepath.Join(dataDir, fileName)
}

// Open opens the database at path, creating it if needed, and applies pending migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	migrations, err := loadMigrations(embeddedMigrations, "migrations")
	if err != nil {
		return nil, err
	}
	return open(ctx, path, migrations)
}

func open(ctx context.Context, path string, migrations []migration) (*sql.DB, error) {
	// Pragmas in the DSN apply to every pooled connection, not just the first.
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	if err := migrate(ctx, db, migrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
