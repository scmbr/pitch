package projects

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/scmbr/pitch/internal/slides"
	"github.com/scmbr/pitch/internal/storage"
)

func projectFixture(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	db, e := storage.Open(context.Background(), storage.Path(dir))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = db.Close() })
	s := NewStore(db, dir)
	s.convert = func(context.Context, string, string) error { return slides.ErrUnavailable }
	return s
}
func pptx(t *testing.T, path, content string) {
	t.Helper()
	out, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(out)
	w, e := z.Create("ppt/slides/slide1.xml")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := w.Write([]byte(content)); e != nil {
		t.Fatal(e)
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	if e := out.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestVersionsUseSnapshotHashAndKeepOldSlides(t *testing.T) {
	s := projectFixture(t)
	source := filepath.Join(t.TempDir(), "talk.pptx")
	pptx(t, source, "first")
	first, e := s.Import(context.Background(), source)
	if e != nil {
		t.Fatal(e)
	}
	if first.Number != 1 || first.Warning != "libreOfficeMissing" {
		t.Fatalf("first: %+v", first)
	}
	unchanged, e := s.Import(context.Background(), source)
	if e != nil || unchanged.ID != first.ID {
		t.Fatal("unchanged file created a new version")
	}
	pdf := filepath.Join(t.TempDir(), "slides.pdf")
	if e := os.WriteFile(pdf, []byte("%PDF-1.7\nfixture"), 0o600); e != nil {
		t.Fatal(e)
	}
	if e := s.AttachPDF(first.ID, pdf); e != nil {
		t.Fatal(e)
	}
	var imageData bytes.Buffer
	if e := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 8, 8))); e != nil {
		t.Fatal(e)
	}
	encoded := base64.StdEncoding.EncodeToString(imageData.Bytes())
	if e := s.CacheSlide(first.ID, 1, encoded); e != nil {
		t.Fatal(e)
	}
	if e := s.CompleteSlides(first.ID, 1); e != nil {
		t.Fatal(e)
	}
	pptx(t, source, "second")
	second, e := s.Import(context.Background(), source)
	if e != nil {
		t.Fatal(e)
	}
	if second.Number != 2 || first.Hash == second.Hash || first.ID == second.ID {
		t.Fatal("changed content not versioned")
	}
	path, e := s.MediaPath(first.ID, 1)
	if e != nil {
		t.Fatal(e)
	}
	old, e := os.ReadFile(path)
	if e != nil || !bytes.Equal(old, imageData.Bytes()) {
		t.Fatal("old slide modified")
	}
	pptx(t, source, "first")
	reverted, e := s.Import(context.Background(), source)
	if e != nil || reverted.ID != first.ID || reverted.SlideCount != 1 {
		t.Fatal("same content should reuse its immutable version")
	}
	if e := os.Remove(source); e != nil {
		t.Fatal(e)
	}
	versions, e := s.Versions(first.ProjectID)
	if e != nil || len(versions) != 2 || !versions[1].HasPDF {
		t.Fatal("cached version unavailable without original")
	}
}
func TestInvalidPresentationAndCache(t *testing.T) {
	s := projectFixture(t)
	source := filepath.Join(t.TempDir(), "bad.pptx")
	if e := os.WriteFile(source, []byte("not a zip"), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Import(context.Background(), source); e == nil {
		t.Fatal("invalid pptx accepted")
	}
	items, e := s.List()
	if e != nil || len(items) != 0 {
		t.Fatal("invalid import created project")
	}
	source = filepath.Join(t.TempDir(), "talk.pdf")
	if e := os.WriteFile(source, []byte("%PDF-1.7\nfixture"), 0o600); e != nil {
		t.Fatal(e)
	}
	v, e := s.Import(context.Background(), source)
	if e != nil || !v.HasPDF {
		t.Fatal("PDF alternative failed")
	}
	if e := s.CompleteSlides(v.ID, 1); e == nil {
		t.Fatal("published missing slide")
	}
	if e := s.CacheSlide(v.ID, 1, "invalid"); e == nil {
		t.Fatal("accepted non-PNG")
	}
	if _, e := s.MediaPath(v.ID, -1); e == nil {
		t.Fatal("accepted invalid page")
	}
}
