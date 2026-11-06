// Package projects snapshots presentations and versions them by SHA-256 content.
package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scmbr/pitch/internal/slides"
)

type Project struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
type Version struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"projectId"`
	Number     int    `json:"number"`
	Hash       string `json:"hash"`
	SlideCount int    `json:"slideCount"`
	HasPDF     bool   `json:"hasPDF"`
	Warning    string `json:"warning"`
}
type Store struct {
	mu      sync.Mutex
	db      *sql.DB
	dir     string
	convert func(context.Context, string, string) error
}

func NewStore(db *sql.DB, dir string) *Store {
	return &Store{db: db, dir: dir, convert: slides.Convert}
}
func (s *Store) List() ([]Project, error) {
	rows, e := s.db.Query("SELECT id,name,pptx_path FROM projects WHERE type='pptx' ORDER BY created_at DESC,id DESC")
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	items := []Project{}
	for rows.Next() {
		var p Project
		if e := rows.Scan(&p.ID, &p.Name, &p.Path); e != nil {
			return nil, e
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (s *Store) Versions(projectID int64) ([]Version, error) {
	rows, e := s.db.Query("SELECT id,project_id,version_number,file_hash,slide_count FROM project_versions WHERE project_id=? ORDER BY version_number DESC", projectID)
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	items := []Version{}
	for rows.Next() {
		var v Version
		if e := rows.Scan(&v.ID, &v.ProjectID, &v.Number, &v.Hash, &v.SlideCount); e != nil {
			return nil, e
		}
		_, e := os.Stat(s.versionFile(v.ID, v.Hash, "document.pdf"))
		v.HasPDF = e == nil
		items = append(items, v)
	}
	return items, rows.Err()
}
func (s *Store) versionFile(id int64, hash, name string) string {
	return filepath.Join(s.dir, "slides", strconv.FormatInt(id, 10), hash, name)
}

func (s *Store) Import(ctx context.Context, source string) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, e := filepath.Abs(source)
	if e != nil {
		return Version{}, e
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".pptx" && ext != ".pdf" {
		return Version{}, errors.New("invalidPresentation")
	}
	if e := os.MkdirAll(filepath.Join(s.dir, "slides"), 0o700); e != nil {
		return Version{}, e
	}
	tmp, e := os.CreateTemp(filepath.Join(s.dir, "slides"), "snapshot-*")
	if e != nil {
		return Version{}, e
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	in, e := os.Open(path)
	if e != nil {
		_ = tmp.Close()
		return Version{}, errors.New("presentationMissing")
	}
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(in, 256*1024*1024+1))
	e = errors.Join(e, in.Close(), tmp.Close())
	if e != nil {
		return Version{}, e
	}
	if n > 256*1024*1024 {
		return Version{}, errors.New("presentationTooLarge")
	}
	if ext == ".pptx" {
		check := tmp.Name() + ".checked"
		defer func() { _ = os.Remove(check) }()
		if e := slides.SanitisePPTX(tmp.Name(), check); e != nil {
			return Version{}, e
		}
	} else {
		probe, e := os.Open(tmp.Name())
		if e != nil {
			return Version{}, e
		}
		var header [5]byte
		_, e = io.ReadFull(probe, header[:])
		_ = probe.Close()
		if e != nil || string(header[:]) != "%PDF-" {
			return Version{}, errors.New("invalidPDF")
		}
	}
	fingerprint := hex.EncodeToString(hash.Sum(nil))
	tx, e := s.db.Begin()
	if e != nil {
		return Version{}, e
	}
	defer func() { _ = tx.Rollback() }()
	var projectID int64
	e = tx.QueryRow("SELECT id FROM projects WHERE type='pptx' AND pptx_path=?", path).Scan(&projectID)
	if errors.Is(e, sql.ErrNoRows) {
		r, err := tx.Exec("INSERT INTO projects(name,type,pptx_path,created_at) VALUES(?,'pptx',?,?)", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), path, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return Version{}, err
		}
		projectID, e = r.LastInsertId()
	}
	if e != nil {
		return Version{}, e
	}
	v := Version{ProjectID: projectID, Hash: fingerprint}
	e = tx.QueryRow("SELECT id,version_number,slide_count FROM project_versions WHERE project_id=? AND file_hash=?", projectID, fingerprint).Scan(&v.ID, &v.Number, &v.SlideCount)
	if errors.Is(e, sql.ErrNoRows) {
		if e = tx.QueryRow("SELECT COALESCE(MAX(version_number),0)+1 FROM project_versions WHERE project_id=?", projectID).Scan(&v.Number); e != nil {
			return Version{}, e
		}
		r, err := tx.Exec("INSERT INTO project_versions(project_id,file_hash,version_number,created_at) VALUES(?,?,?,?)", projectID, fingerprint, v.Number, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return Version{}, err
		}
		v.ID, e = r.LastInsertId()
	}
	if e != nil {
		return Version{}, e
	}
	folder := filepath.Dir(s.versionFile(v.ID, v.Hash, "source"))
	if e := os.MkdirAll(folder, 0o700); e != nil {
		return Version{}, e
	}
	snapshot := s.versionFile(v.ID, v.Hash, "source"+ext)
	if _, e := os.Stat(snapshot); errors.Is(e, os.ErrNotExist) {
		if e := copyFile(tmp.Name(), snapshot); e != nil {
			return Version{}, e
		}
	} else if e != nil {
		return Version{}, e
	}
	if e := tx.Commit(); e != nil {
		return Version{}, e
	}
	pdf := s.versionFile(v.ID, v.Hash, "document.pdf")
	if _, e := os.Stat(pdf); e == nil {
		v.HasPDF = true
		return v, nil
	}
	if ext == ".pdf" {
		e = copyPDF(snapshot, pdf)
	} else {
		e = s.convert(ctx, snapshot, pdf)
	}
	if e != nil {
		v.Warning = "conversionFailed"
		if errors.Is(e, slides.ErrUnavailable) {
			v.Warning = "libreOfficeMissing"
		}
		return v, nil
	}
	v.HasPDF = true
	return v, nil
}

func copyFile(source, destination string) error {
	in, e := os.Open(source)
	if e != nil {
		return e
	}
	defer func() { _ = in.Close() }()
	if _, e := os.Stat(destination); e == nil {
		return os.ErrExist
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	out, e := os.CreateTemp(filepath.Dir(destination), "copy-*")
	if e != nil {
		return e
	}
	defer func() { _ = os.Remove(out.Name()) }()
	_, e = io.Copy(out, in)
	e = errors.Join(e, out.Sync(), out.Close())
	if e != nil {
		return e
	}
	return os.Rename(out.Name(), destination)
}
func copyPDF(source, destination string) error {
	in, e := os.Open(source)
	if e != nil {
		return e
	}
	var header [5]byte
	_, e = io.ReadFull(in, header[:])
	_ = in.Close()
	if e != nil || string(header[:]) != "%PDF-" {
		return errors.New("invalidPDF")
	}
	f, e := os.Stat(source)
	if e != nil {
		return e
	}
	if f.Size() > 256*1024*1024 {
		return errors.New("presentationTooLarge")
	}
	return copyFile(source, destination)
}
func (s *Store) AttachPDF(versionID int64, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	destination, e := s.MediaPath(versionID, 0)
	if e != nil {
		return e
	}
	return copyPDF(path, destination)
}

// Page zero is the PDF; positive indexes address cached PNG slide images.
func (s *Store) MediaPath(versionID int64, page int) (string, error) {
	if versionID <= 0 || page < 0 || page > 500 {
		return "", errors.New("invalidSlide")
	}
	var hash string
	if e := s.db.QueryRow("SELECT file_hash FROM project_versions WHERE id=?", versionID).Scan(&hash); e != nil {
		return "", e
	}
	decoded, e := hex.DecodeString(hash)
	if e != nil || len(decoded) != 32 {
		return "", errors.New("invalidPresentation")
	}
	name := "document.pdf"
	if page > 0 {
		name = strconv.Itoa(page) + ".png"
	}
	return s.versionFile(versionID, hash, name), nil
}
func (s *Store) CacheSlide(versionID int64, page int, encoded string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if page < 1 || len(encoded) > 8*1024*1024 {
		return errors.New("invalidSlide")
	}
	path, e := s.MediaPath(versionID, page)
	if e != nil {
		return e
	}
	if _, e := os.Stat(path); e == nil {
		return nil
	}
	data, e := base64.StdEncoding.DecodeString(encoded)
	if e != nil {
		return e
	}
	config, e := png.DecodeConfig(bytes.NewReader(data))
	if e != nil || config.Width > 1920 || config.Height > 1920 {
		return errors.New("invalidSlide")
	}
	tmp, e := os.CreateTemp(filepath.Dir(path), "slide-*")
	if e != nil {
		return e
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, e = tmp.Write(data)
	e = errors.Join(e, tmp.Sync(), tmp.Close())
	if e != nil {
		return e
	}
	return os.Rename(tmp.Name(), path)
}
func (s *Store) CompleteSlides(versionID int64, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if count < 1 || count > 500 {
		return errors.New("invalidSlide")
	}
	for page := 1; page <= count; page++ {
		path, e := s.MediaPath(versionID, page)
		if e != nil {
			return e
		}
		if _, e := os.Stat(path); e != nil {
			return e
		}
	}
	_, e := s.db.Exec("UPDATE project_versions SET slide_count=? WHERE id=?", count, versionID)
	return e
}
