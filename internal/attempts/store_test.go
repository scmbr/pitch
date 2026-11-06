package attempts

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scmbr/pitch/internal/settings"
	"github.com/scmbr/pitch/internal/storage"
)

func fixture(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), storage.Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewStore(db, dir, settings.NewStore(filepath.Join(dir, "settings.json"), settings.Settings{DataDir: dir, Language: "en"}))
}
func video(t *testing.T, s *Store, token string, size int) {
	t.Helper()
	p, e := s.MediaPath(token, false)
	if e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(filepath.Dir(p), 0o700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, make([]byte, size), 0o600); e != nil {
		t.Fatal(e)
	}
}
func list(t *testing.T, s *Store) []Attempt {
	t.Helper()
	rows, e := s.List()
	if e != nil {
		t.Fatal(e)
	}
	return rows
}

func TestAttemptRatingsThumbnailAndDeletion(t *testing.T) {
	s := fixture(t)
	token := strings.Repeat("a", 32)
	video(t, s, token, 100)
	var thumbnail bytes.Buffer
	if e := jpeg.Encode(&thumbnail, image.NewRGBA(image.Rect(0, 0, 320, 180)), nil); e != nil {
		t.Fatal(e)
	}
	if e := s.Add(token, 4.5, base64.StdEncoding.EncodeToString(thumbnail.Bytes())); e != nil {
		t.Fatal(e)
	}
	a := list(t, s)[0]
	if !a.Thumbnail || a.Duration != 4.5 {
		t.Fatalf("metadata: %+v", a)
	}
	if e := s.Update(a.ID, Scores{1, 4, 5}, "my note"); e != nil {
		t.Fatal(e)
	}
	if e := s.Update(a.ID, Scores{0, 6, 1}, "invalid"); e == nil {
		t.Fatal("accepted invalid score")
	}
	a = list(t, s)[0]
	if a.Note != "my note" || a.Scores.Confidence != 4 {
		t.Fatal("ratings not persisted")
	}
	p, _ := s.MediaPath(token, true)
	got, e := os.ReadFile(p)
	if e != nil || !bytes.Equal(got, thumbnail.Bytes()) {
		t.Fatal("cached thumbnail changed")
	}
	if e := s.Delete(a.ID); e != nil {
		t.Fatal(e)
	}
	if len(list(t, s)) != 0 {
		t.Fatal("row survived deletion")
	}
	for _, thumb := range []bool{false, true} {
		p, _ := s.MediaPath(token, thumb)
		if _, e := os.Stat(p); !os.IsNotExist(e) {
			t.Fatal("media survived deletion")
		}
	}
}

func TestStoragePolicyDoesNotDeleteWithoutOptIn(t *testing.T) {
	s := fixture(t)
	old, newer := strings.Repeat("a", 32), strings.Repeat("b", 32)
	p, _ := s.settings.Load()
	p.StorageLimitMB = 1
	if e := s.settings.Save(p); e != nil {
		t.Fatal(e)
	}
	video(t, s, old, 600*1024)
	if e := s.Add(old, 1, ""); e != nil {
		t.Fatal(e)
	}
	video(t, s, newer, 600*1024)
	if e := s.Add(newer, 1, ""); e == nil || !strings.Contains(e.Error(), "storageLimit") {
		t.Fatalf("limit: %v", e)
	}
	if len(list(t, s)) != 1 {
		t.Fatal("deleted old attempt without consent")
	}
	p.AutoDelete = true
	if e := s.settings.Save(p); e != nil {
		t.Fatal(e)
	}
	// Invalid drafts cannot trigger automatic deletion.
	if e := s.Add(newer, 1, "not jpeg"); e == nil {
		t.Fatal("invalid thumbnail accepted")
	}
	if len(list(t, s)) != 1 {
		t.Fatal("invalid draft deleted old attempt")
	}
	if e := s.Add(newer, 1, ""); e != nil {
		t.Fatal(e)
	}
	rows := list(t, s)
	if len(rows) != 1 || rows[0].Token != newer {
		t.Fatal("did not replace oldest attempt")
	}
	path, _ := s.MediaPath(old, false)
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("old video retained")
	}
}

func TestInterruptedDeletionAndOrphanRecovery(t *testing.T) {
	s := fixture(t)
	token := strings.Repeat("c", 32)
	video(t, s, token, 10)
	if e := s.Add(token, 1, ""); e != nil {
		t.Fatal(e)
	}
	id := list(t, s)[0].ID
	path, _ := s.MediaPath(token, false)
	if e := os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(path, 0o700); e != nil {
		t.Fatal(e)
	}
	child := filepath.Join(path, "busy")
	if e := os.WriteFile(child, []byte("x"), 0o600); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete(id); e == nil {
		t.Fatal("expected file removal failure")
	}
	if rows := list(t, s); len(rows) != 1 || !rows[0].Deleting {
		t.Fatal("lost deletion intent")
	}
	if e := os.Remove(child); e != nil {
		t.Fatal(e)
	}
	if e := s.Recover(); e != nil {
		t.Fatal(e)
	}
	if len(list(t, s)) != 0 {
		t.Fatal("deletion not resumed")
	}
	video(t, s, token, 10)
	if e := s.Recover(); e != nil {
		t.Fatal(e)
	}
	if e := s.Recover(); e != nil {
		t.Fatal(e)
	}
	if len(list(t, s)) != 1 {
		t.Fatal("recovery failed or duplicated recording")
	}
	if _, e := s.MediaPath("../outside", false); e == nil {
		t.Fatal("accepted arbitrary path")
	}
}

func TestPresentationAttemptPersistsValidatedTimeline(t *testing.T) {
	s := fixture(t)
	for _, sql := range []string{
		"INSERT INTO projects(id,name,type,created_at) VALUES(10,'Talk','pptx','now')",
		"INSERT INTO project_versions(id,project_id,file_hash,version_number,created_at,slide_count) VALUES(20,10,'hash',2,'now',2)",
	} {
		if _, e := s.db.Exec(sql); e != nil {
			t.Fatal(e)
		}
	}
	token := strings.Repeat("d", 32)
	video(t, s, token, 10)
	draft := Draft{Duration: 5, ProjectID: 10, VersionID: 20, Timings: []SlideTiming{{0, 0, 2}, {1, 2, 5}}}
	if e := s.AddDraft(token, draft); e != nil {
		t.Fatal(e)
	}
	a := list(t, s)[0]
	if a.VersionID != 20 || a.ProjectName != "Talk" || a.VersionNumber != 2 {
		t.Fatalf("version: %+v", a)
	}
	timings, e := s.Timings(a.ID)
	if e != nil || len(timings) != 2 || timings[1].Start != 2 {
		t.Fatalf("timeline: %+v %v", timings, e)
	}
	if e := s.Delete(a.ID); e != nil {
		t.Fatal(e)
	}
	timings, e = s.Timings(a.ID)
	if e != nil || len(timings) != 0 {
		t.Fatal("timeline survived deletion")
	}
	for _, invalid := range []Draft{
		{Duration: 5, ProjectID: 11, VersionID: 20},
		{Duration: 5, ProjectID: 10, VersionID: 20, Timings: []SlideTiming{{2, 0, 5}}},
		{Duration: 5, ProjectID: 10, VersionID: 20, Timings: []SlideTiming{{0, 0, 4}, {1, 2, 5}}},
		{Duration: 5, ProjectID: 10, VersionID: 20, Timings: []SlideTiming{{0, 0, 6}}},
	} {
		video(t, s, token, 10)
		if e := s.AddDraft(token, invalid); e == nil {
			t.Fatalf("accepted invalid draft: %+v", invalid)
		}
	}
}
