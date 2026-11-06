package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.sqlite")
}

func mustOpen(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestOpenCreatesSchema(t *testing.T) {
	db := mustOpen(t, tempDBPath(t))

	for _, table := range []string{"projects", "project_versions", "attempts", "slide_timings", "schema_migrations"} {
		if n := count(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table); n != 1 {
			t.Errorf("table %s missing", table)
		}
	}
	if v := count(t, db, `SELECT MAX(version) FROM schema_migrations`); v != 2 {
		t.Errorf("schema version = %d, want 2", v)
	}
}

func TestOpenTwiceKeepsDataAndDoesNotReapply(t *testing.T) {
	path := tempDBPath(t)
	db := mustOpen(t, path)
	mustExec(t, db, `INSERT INTO projects (name, type, created_at) VALUES ('keep me', 'free', '2026-01-01T00:00:00Z')`)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db = mustOpen(t, path)
	if n := count(t, db, `SELECT COUNT(*) FROM projects`); n != 1 {
		t.Errorf("projects = %d, want 1", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM schema_migrations`); n != 2 {
		t.Errorf("migration rows = %d, want 2", n)
	}
}

func TestForeignKeysAreEnforcedAndCascade(t *testing.T) {
	db := mustOpen(t, tempDBPath(t))

	if _, err := db.Exec(`INSERT INTO attempts (project_id, video_path, duration_sec, created_at)
		VALUES (999, 'v.webm', 1, '2026-01-01T00:00:00Z')`); err == nil {
		t.Error("attempt for a missing project was accepted")
	}

	mustExec(t, db, `INSERT INTO projects (id, name, type, created_at) VALUES (1, 'p', 'pptx', '2026-01-01T00:00:00Z')`)
	mustExec(t, db, `INSERT INTO attempts (id, project_id, video_path, duration_sec, created_at)
		VALUES (1, 1, 'v.webm', 12.5, '2026-01-01T00:00:00Z')`)
	mustExec(t, db, `INSERT INTO slide_timings (attempt_id, slide_index, start_sec, end_sec) VALUES (1, 0, 0, 5)`)

	mustExec(t, db, `DELETE FROM projects WHERE id = 1`)
	if n := count(t, db, `SELECT COUNT(*) FROM attempts`); n != 0 {
		t.Errorf("attempts after deleting project = %d, want 0", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM slide_timings`); n != 0 {
		t.Errorf("slide_timings after deleting project = %d, want 0", n)
	}
}

func TestConstraintsRejectBadRows(t *testing.T) {
	db := mustOpen(t, tempDBPath(t))
	mustExec(t, db, `INSERT INTO projects (id, name, type, created_at) VALUES (1, 'p', 'pptx', '2026-01-01T00:00:00Z')`)
	mustExec(t, db, `INSERT INTO project_versions (project_id, file_hash, version_number, created_at) VALUES (1, 'abc', 1, '2026-01-01T00:00:00Z')`)
	mustExec(t, db, `INSERT INTO attempts (id, project_id, video_path, duration_sec, created_at) VALUES (1, 1, 'v.webm', 1, '2026-01-01T00:00:00Z')`)

	tests := []struct {
		name  string
		query string
	}{
		{"unknown project type", `INSERT INTO projects (name, type, created_at) VALUES ('x', 'other', '2026-01-01T00:00:00Z')`},
		{"duplicate version number", `INSERT INTO project_versions (project_id, file_hash, version_number, created_at) VALUES (1, 'def', 1, '2026-01-01T00:00:00Z')`},
		{"duplicate file hash", `INSERT INTO project_versions (project_id, file_hash, version_number, created_at) VALUES (1, 'abc', 2, '2026-01-01T00:00:00Z')`},
		{"negative duration", `INSERT INTO attempts (project_id, video_path, duration_sec, created_at) VALUES (1, 'v.webm', -1, '2026-01-01T00:00:00Z')`},
		{"slide ends before it starts", `INSERT INTO slide_timings (attempt_id, slide_index, start_sec, end_sec) VALUES (1, 0, 5, 4)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := db.Exec(tt.query); err == nil {
				t.Error("row was accepted")
			}
		})
	}
}

func openWith(t *testing.T, path string, migrations []migration) (*sql.DB, error) {
	t.Helper()
	db, err := open(context.Background(), path, migrations)
	if err == nil {
		t.Cleanup(func() { _ = db.Close() })
	}
	return db, err
}

func TestMigrationsApplyIncrementallyAndKeepData(t *testing.T) {
	path := tempDBPath(t)
	first := migration{version: 1, name: "one", sql: `CREATE TABLE a (v TEXT); INSERT INTO a VALUES ('old');`}
	second := migration{version: 2, name: "two", sql: `ALTER TABLE a ADD COLUMN w TEXT;`}

	db, err := openWith(t, path, []migration{first})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = openWith(t, path, []migration{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM a WHERE v = 'old'`); n != 1 {
		t.Errorf("old row lost, count = %d", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM pragma_table_info('a') WHERE name = 'w'`); n != 1 {
		t.Error("second migration was not applied")
	}
}

func TestFailedMigrationRollsBackCompletely(t *testing.T) {
	path := tempDBPath(t)
	good := migration{version: 1, name: "good", sql: `CREATE TABLE a (v TEXT);`}
	bad := migration{version: 2, name: "bad", sql: `CREATE TABLE half_done (v TEXT); THIS IS NOT SQL;`}

	if _, err := openWith(t, path, []migration{good, bad}); err == nil {
		t.Fatal("broken migration was accepted")
	}

	db, err := openWith(t, path, []migration{good})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'half_done'`); n != 0 {
		t.Error("partial migration was not rolled back")
	}
	if v := count(t, db, `SELECT MAX(version) FROM schema_migrations`); v != 1 {
		t.Errorf("schema version = %d, want 1", v)
	}
}

func TestDatabaseFromNewerAppIsRejected(t *testing.T) {
	path := tempDBPath(t)
	one := migration{version: 1, name: "one", sql: `CREATE TABLE a (v TEXT);`}
	two := migration{version: 2, name: "two", sql: `CREATE TABLE b (v TEXT);`}

	db, err := openWith(t, path, []migration{one, two})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := openWith(t, path, []migration{one}); !errors.Is(err, ErrDatabaseNewer) {
		t.Errorf("error = %v, want ErrDatabaseNewer", err)
	}
}

func TestLoadMigrations(t *testing.T) {
	file := func(body string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(body)} }

	t.Run("sorts by version", func(t *testing.T) {
		fsys := fstest.MapFS{
			"m/0002_second.sql": file("SELECT 2;"),
			"m/0001_first.sql":  file("SELECT 1;"),
		}
		got, err := loadMigrations(fsys, "m")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].name != "first" || got[1].name != "second" || got[1].sql != "SELECT 2;" {
			t.Errorf("got %+v", got)
		}
	})

	bad := map[string]fstest.MapFS{
		"bad file name": {"m/first.sql": file("")},
		"gap":           {"m/0001_a.sql": file(""), "m/0003_c.sql": file("")},
		"duplicate":     {"m/0001_a.sql": file(""), "m/0001_b.sql": file("")},
		"starts at two": {"m/0002_a.sql": file("")},
	}
	for name, fsys := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := loadMigrations(fsys, "m"); err == nil {
				t.Error("invalid migration set was accepted")
			}
		})
	}
}

func TestEmbeddedMigrationsAreValid(t *testing.T) {
	got, err := loadMigrations(embeddedMigrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Error("no embedded migrations")
	}
}
