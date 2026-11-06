package app

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// RecordingService accepts ordered, bounded chunks. Only its own active token
// can write a file; callers never supply filesystem paths.
type RecordingService struct {
	mu    sync.Mutex
	dir   string
	token string
	file  *os.File
	bytes int64
}

func NewRecordingService(dataDir string) *RecordingService {
	return &RecordingService{dir: filepath.Join(dataDir, "videos")}
}

func (s *RecordingService) Begin() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" {
		return "", errors.New("recordingActive")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(id[:])
	f, err := os.OpenFile(filepath.Join(s.dir, token+".partial"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	s.token, s.file, s.bytes = token, f, 0
	return token, nil
}

func (s *RecordingService) Append(token, encoded string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil || token != s.token {
		return errors.New("recordingUnknown")
	}
	if len(encoded) > 512*1024 {
		return errors.New("recordingChunkTooLarge")
	}
	chunk, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	n, err := s.file.Write(chunk)
	s.bytes += int64(n)
	return err
}

// Finish publishes only a nonempty, flushed recording. Its identifier can later
// be attached to an attempt without passing an arbitrary local path over IPC.
func (s *RecordingService) Finish(token string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil || token != s.token {
		return "", errors.New("recordingUnknown")
	}
	if s.bytes == 0 {
		return "", errors.New("recordingEmpty")
	}
	if err := s.file.Sync(); err != nil {
		return "", err
	}
	if err := s.file.Close(); err != nil {
		s.file = nil
		return "", err
	}
	s.file = nil
	if err := os.Rename(filepath.Join(s.dir, token+".partial"), filepath.Join(s.dir, token+".webm")); err != nil {
		return "", err
	}
	s.token = ""
	return token, nil
}

func (s *RecordingService) Abort(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == "" || token != s.token {
		return nil
	}
	return s.abort()
}

func (s *RecordingService) abort() error {
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	path := filepath.Join(s.dir, s.token+".partial")
	s.token = ""
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *RecordingService) ServiceShutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		return nil
	}
	return s.abort()
}
