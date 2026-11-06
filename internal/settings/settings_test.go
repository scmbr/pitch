package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg", fileName)
	defaults := Settings{DataDir: filepath.Join(dir, "default-data")}
	return NewStore(path, defaults), path
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	store, _ := newTestStore(t)

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != store.defaults {
		t.Errorf("got %+v, want defaults %+v", got, store.defaults)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	store, _ := newTestStore(t)
	want := Settings{DataDir: filepath.Join(t.TempDir(), "my-data"), Language: LanguageRU}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSaveOverwritesAndLeavesNoTempFiles(t *testing.T) {
	store, path := newTestStore(t)
	first := Settings{DataDir: filepath.Join(t.TempDir(), "a"), Language: LanguageAuto}
	second := Settings{DataDir: filepath.Join(t.TempDir(), "b"), Language: LanguageEN}

	for _, v := range []Settings{first, second} {
		if err := store.Save(v); err != nil {
			t.Fatalf("Save(%+v): %v", v, err)
		}
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != second {
		t.Errorf("got %+v, want %+v", got, second)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("config dir has %d entries, want only the settings file", len(entries))
	}
}

func TestSaveRejectsInvalidSettingsAndKeepsOldFile(t *testing.T) {
	store, _ := newTestStore(t)
	good := Settings{DataDir: filepath.Join(t.TempDir(), "ok"), Language: LanguageAuto}
	if err := store.Save(good); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		in   Settings
		want error
	}{
		{"empty", Settings{}, ErrDataDirEmpty},
		{"relative", Settings{DataDir: filepath.Join("some", "dir")}, ErrDataDirNotAbsolute},
		{"unknown language", Settings{DataDir: good.DataDir, Language: "de"}, ErrLanguageUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.Save(tt.in); !errors.Is(err, tt.want) {
				t.Errorf("Save error = %v, want %v", err, tt.want)
			}
		})
	}

	got, err := store.Load()
	if err != nil || got != good {
		t.Errorf("after rejected saves got %+v, %v; want %+v", got, err, good)
	}
}

func TestLoadCorruptFileReturnsDefaultsAndKeepsFile(t *testing.T) {
	store, path := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	const garbage = "{not json"
	if err := os.WriteFile(path, []byte(garbage), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("error = %v, want ErrCorrupt", err)
	}
	if got != store.defaults {
		t.Errorf("got %+v, want defaults", got)
	}
	data, _ := os.ReadFile(path)
	if string(data) != garbage {
		t.Errorf("corrupt file was modified: %q", data)
	}
}

func TestLoadFillsEmptyDataDirFromDefaultsAndIgnoresUnknownFields(t *testing.T) {
	store, path := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"dataDir":"","futureField":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != store.defaults {
		t.Errorf("got %+v, want defaults %+v", got, store.defaults)
	}
}

func TestEmptyLanguageMeansAuto(t *testing.T) {
	store, path := newTestStore(t)
	if err := store.Save(Settings{DataDir: store.defaults.DataDir}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Language != LanguageAuto {
		t.Errorf("language = %q, want %q", got.Language, LanguageAuto)
	}

	if err := os.WriteFile(path, []byte(`{"dataDir":"`+filepath.ToSlash(store.defaults.DataDir)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = store.Load()
	if err != nil || got.Language != LanguageAuto {
		t.Errorf("old file without language: got %+v, %v", got, err)
	}
}

func TestPrepareDataDir(t *testing.T) {
	t.Run("creates missing folder", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "a", "b")
		if err := PrepareDataDir(dir); err != nil {
			t.Fatalf("PrepareDataDir: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("probe file left behind: %v", entries)
		}
	})

	t.Run("rejects relative path", func(t *testing.T) {
		if err := PrepareDataDir("relative"); !errors.Is(err, ErrDataDirNotAbsolute) {
			t.Errorf("error = %v, want ErrDataDirNotAbsolute", err)
		}
	})

	t.Run("preserves existing files", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, probeName)
		const content = "keep this file"
		if err := os.WriteFile(existing, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := PrepareDataDir(dir); err != nil {
			t.Fatalf("PrepareDataDir: %v", err)
		}
		got, err := os.ReadFile(existing)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Errorf("existing file = %q, want %q", got, content)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != probeName {
			t.Errorf("unexpected directory contents: %v", entries)
		}
	})

	t.Run("rejects a path blocked by a file", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := PrepareDataDir(filepath.Join(blocker, "sub")); !errors.Is(err, ErrDataDirNotWritable) {
			t.Errorf("error = %v, want ErrDataDirNotWritable", err)
		}
	})
}
