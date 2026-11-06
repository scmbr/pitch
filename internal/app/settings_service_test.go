package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scmbr/pitch/internal/settings"
)

func newService(t *testing.T) (*SettingsService, string) {
	t.Helper()
	root := t.TempDir()
	active := filepath.Join(root, "active")
	store := settings.NewStore(filepath.Join(root, "cfg", "settings.json"), settings.Settings{DataDir: active})
	pick := func(string, string) (string, error) { return filepath.Join(root, "picked"), nil }
	return NewSettingsService(store, active, pick), root
}

func TestGetDefaultsNeedNoRestart(t *testing.T) {
	svc, _ := newService(t)

	view, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.DataDir != view.ActiveDataDir || view.RestartRequired {
		t.Errorf("fresh settings should match the active folder: %+v", view)
	}
	if view.Language != settings.LanguageAuto || view.Warning != "" {
		t.Errorf("unexpected defaults: %+v", view)
	}
}

func TestSaveNewFolderCreatesItAndAsksForRestart(t *testing.T) {
	svc, root := newService(t)
	next := filepath.Join(root, "elsewhere")

	res := svc.Save(next, settings.LanguageEN)
	if !res.OK || res.ErrorCode != "" {
		t.Fatalf("Save: %+v", res)
	}
	if info, err := os.Stat(next); err != nil || !info.IsDir() {
		t.Errorf("new data folder was not created: %v", err)
	}

	view, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if view.DataDir != next || view.Language != settings.LanguageEN {
		t.Errorf("saved values not returned: %+v", view)
	}
	if !view.RestartRequired || view.ActiveDataDir == next {
		t.Errorf("changed folder must require a restart: %+v", view)
	}
}

func TestSaveReportsCodes(t *testing.T) {
	svc, root := newService(t)
	blocker := filepath.Join(root, "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		dataDir  string
		language string
		want     string
	}{
		{"empty folder", "", settings.LanguageAuto, CodeDataDirEmpty},
		{"relative folder", "relative", settings.LanguageAuto, CodeDataDirNotAbsolute},
		{"folder under a file", filepath.Join(blocker, "sub"), settings.LanguageAuto, CodeDataDirNotWritable},
		{"unknown language", filepath.Join(root, "ok"), "de", CodeLanguageUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := svc.Save(tt.dataDir, tt.language)
			if res.OK || res.ErrorCode != tt.want {
				t.Errorf("got %+v, want code %q", res, tt.want)
			}
		})
	}
}

func TestRejectedSaveKeepsPreviousSettings(t *testing.T) {
	svc, root := newService(t)
	good := filepath.Join(root, "good")
	if res := svc.Save(good, settings.LanguageRU); !res.OK {
		t.Fatalf("Save: %+v", res)
	}

	if res := svc.Save("relative", settings.LanguageEN); res.OK {
		t.Fatal("invalid save was accepted")
	}
	view, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if view.DataDir != good || view.Language != settings.LanguageRU {
		t.Errorf("previous settings lost: %+v", view)
	}
}

func TestGetWarnsAboutCorruptFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(root, "data")
	svc := NewSettingsService(settings.NewStore(path, settings.Settings{DataDir: active}), active, nil)

	view, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.Warning != CodeSettingsCorrupt || view.DataDir != active {
		t.Errorf("got %+v", view)
	}
}

func TestPickDataDirPassesStartFolder(t *testing.T) {
	root := t.TempDir()
	var gotStart, gotTitle string
	pick := func(start, title string) (string, error) { gotStart, gotTitle = start, title; return "", nil }
	svc := NewSettingsService(settings.NewStore(filepath.Join(root, "s.json"), settings.Settings{DataDir: root}), root, pick)

	chosen, err := svc.PickDataDir("start-here", "Choose a folder")
	if err != nil || chosen != "" {
		t.Errorf("got %q, %v; cancelling should return an empty path", chosen, err)
	}
	if gotStart != "start-here" || gotTitle != "Choose a folder" {
		t.Errorf("start folder = %q, title = %q", gotStart, gotTitle)
	}
}
