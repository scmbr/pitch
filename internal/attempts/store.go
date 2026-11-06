// Package attempts owns attempt metadata and its associated local media files.
package attempts

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scmbr/pitch/internal/settings"
)

type Scores struct {
	Structure  int `json:"structure"`
	Confidence int `json:"confidence"`
	Timing     int `json:"timing"`
}

type Attempt struct {
	ID        int64   `json:"id"`
	ProjectID int64   `json:"projectId"`
	VersionID int64   `json:"versionId"`
	Token     string  `json:"token"`
	Duration  float64 `json:"duration"`
	Topic     string  `json:"topic"`
	Scores    Scores  `json:"scores"`
	Note      string  `json:"note"`
	CreatedAt string  `json:"createdAt"`
	Thumbnail bool    `json:"thumbnail"`
	Deleting  bool    `json:"deleting"`
}

type Store struct {
	mu       sync.Mutex
	db       *sql.DB
	dir      string
	settings *settings.Store
}

func NewStore(db *sql.DB, dir string, preferences *settings.Store) *Store {
	return &Store{db: db, dir: dir, settings: preferences}
}

func ValidToken(token string) bool {
	b, err := hex.DecodeString(token)
	return err == nil && len(b) == 16 && token == hex.EncodeToString(b)
}

// MediaPath accepts only generated identifiers, never relative or absolute paths.
func (s *Store) MediaPath(token string, thumbnail bool) (string, error) {
	if !ValidToken(token) {
		return "", errors.New("invalidRecording")
	}
	if thumbnail {
		return filepath.Join(s.dir, "thumbs", token+".jpg"), nil
	}
	return filepath.Join(s.dir, "videos", token+".webm"), nil
}

func (s *Store) Add(token string, duration float64, thumbnail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration < 0 {
		return errors.New("invalidDuration")
	}
	path, err := s.MediaPath(token, false)
	if err != nil {
		return err
	}
	file, err := os.Stat(path)
	if err != nil {
		return err
	}
	var exists bool
	if err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM attempts WHERE video_path=?)", token+".webm").Scan(&exists); err != nil {
		return err
	}
	if exists {
		return errors.New("recordingAlreadySaved")
	}
	thumb := ""
	var thumbData []byte
	if thumbnail != "" {
		if len(thumbnail) > 512*1024 {
			return errors.New("invalidThumbnail")
		}
		data, err := base64.StdEncoding.DecodeString(thumbnail)
		if err != nil {
			return err
		}
		config, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width > 1280 || config.Height > 720 {
			return errors.New("invalidThumbnail")
		}
		thumbData = data
	}
	if err := s.makeSpace(file.Size() + int64(len(thumbData))); err != nil {
		return err
	}
	if len(thumbData) != 0 {
		thumb, _ = s.MediaPath(token, true)
		if err := os.MkdirAll(filepath.Dir(thumb), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(thumb, thumbData, 0o600); err != nil {
			return err
		}
	}
	if err := s.insert(token, duration, thumb != ""); err != nil {
		if thumb != "" {
			_ = os.Remove(thumb)
		}
		return err
	}
	return nil
}

func (s *Store) insert(token string, duration float64, thumbnail bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var projectID int64
	err = tx.QueryRow("SELECT id FROM projects WHERE type = 'free' ORDER BY id LIMIT 1").Scan(&projectID)
	if errors.Is(err, sql.ErrNoRows) {
		r, e := tx.Exec("INSERT INTO projects(name,type,created_at) VALUES('free','free',?)", time.Now().UTC().Format(time.RFC3339Nano))
		if e != nil {
			return e
		}
		projectID, err = r.LastInsertId()
	}
	if err != nil {
		return err
	}
	thumb := ""
	if thumbnail {
		thumb = token + ".jpg"
	}
	_, err = tx.Exec("INSERT INTO attempts(project_id,video_path,thumb_path,duration_sec,topic,scores,note,created_at) VALUES(?,?,?,?,?,?,?,?)", projectID, token+".webm", thumb, duration, "", "{}", "", time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) List() ([]Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *Store) list() ([]Attempt, error) {
	rows, err := s.db.Query("SELECT id,project_id,COALESCE(version_id,0),video_path,duration_sec,COALESCE(topic,''),COALESCE(scores,'{}'),COALESCE(note,''),created_at,COALESCE(thumb_path,''),deleting FROM attempts ORDER BY created_at DESC,id DESC")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []Attempt{}
	for rows.Next() {
		var a Attempt
		var name, score, thumb string
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.VersionID, &name, &a.Duration, &a.Topic, &score, &a.Note, &a.CreatedAt, &thumb, &a.Deleting); err != nil {
			return nil, err
		}
		a.Token = name[:len(name)-len(filepath.Ext(name))]
		if !ValidToken(a.Token) {
			return nil, errors.New("invalidRecording")
		}
		if err := json.Unmarshal([]byte(score), &a.Scores); err != nil {
			return nil, err
		}
		a.Thumbnail = thumb != ""
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *Store) Update(id int64, scores Scores, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range []int{scores.Structure, scores.Confidence, scores.Timing} {
		if n < 1 || n > 5 {
			return errors.New("invalidScores")
		}
	}
	if len(note) > 20000 {
		return errors.New("noteTooLong")
	}
	encoded, err := json.Marshal(scores)
	if err != nil {
		return err
	}
	r, err := s.db.Exec("UPDATE attempts SET scores=?,note=? WHERE id=? AND deleting=0", string(encoded), note, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n != 1 {
		return errors.New("attemptUnknown")
	}
	return err
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.remove(id)
}

// Persist intent first. A file lock or interruption leaves an identifiable,
// retryable row; metadata is removed only after both media files are gone.
func (s *Store) remove(id int64) error {
	var name string
	err := s.db.QueryRow("SELECT video_path FROM attempts WHERE id=?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	token := name[:len(name)-len(filepath.Ext(name))]
	if !ValidToken(token) {
		return errors.New("invalidRecording")
	}
	if _, err := s.db.Exec("UPDATE attempts SET deleting=1 WHERE id=?", id); err != nil {
		return err
	}
	for _, thumbnail := range []bool{false, true} {
		path, _ := s.MediaPath(token, thumbnail)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	_, err = s.db.Exec("DELETE FROM attempts WHERE id=?", id)
	return err
}

func (s *Store) makeSpace(incoming int64) error {
	preferences, err := s.settings.Load()
	if err != nil && !errors.Is(err, settings.ErrCorrupt) {
		return err
	}
	if preferences.StorageLimitMB <= 0 {
		return nil
	}
	limit := preferences.StorageLimitMB * 1024 * 1024
	if incoming > limit {
		return errors.New("storageLimit")
	}
	items, err := s.list()
	if err != nil {
		return err
	}
	var usage int64
	sizes := map[int64]int64{}
	for _, a := range items {
		for _, thumbnail := range []bool{false, true} {
			p, _ := s.MediaPath(a.Token, thumbnail)
			f, e := os.Stat(p)
			if e == nil {
				sizes[a.ID] += f.Size()
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
		usage += sizes[a.ID]
	}
	for i := len(items) - 1; usage+incoming > limit && i >= 0; i-- {
		if !preferences.AutoDelete {
			return errors.New("storageLimit")
		}
		if err := s.remove(items[i].ID); err != nil {
			return err
		}
		usage -= sizes[items[i].ID]
	}
	return nil
}

// Recover imports finalized files left before metadata was committed (including
// recordings from older versions), and resumes interrupted deletion intents.
func (s *Store) Recover() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.list()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, a := range items {
		known[a.Token] = true
		if a.Deleting {
			if err := s.remove(a.ID); err != nil {
				return fmt.Errorf("recover deletion: %w", err)
			}
		}
	}
	files, err := os.ReadDir(filepath.Join(s.dir, "videos"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".webm" {
			continue
		}
		token := file.Name()[:len(file.Name())-5]
		if !ValidToken(token) || known[token] {
			continue
		}
		if err := s.insert(token, 0, false); err != nil {
			return err
		}
	}
	return nil
}
