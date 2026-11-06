package app

import (
	"errors"
	"github.com/scmbr/pitch/internal/topics"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TopicService struct {
	store *topics.Store
	pick  func(string, string) (string, error)
}

func NewTopicService(store *topics.Store, pick func(string, string) (string, error)) *TopicService {
	return &TopicService{store: store, pick: pick}
}
func (s *TopicService) Sets() ([]topics.Set, error) { return s.store.Sets() }
func (s *TopicService) Draw(ids []int64, avoid int) (topics.Topic, error) {
	return s.store.Draw(ids, avoid)
}
func (s *TopicService) Import(title string) (topics.Set, error) {
	path, e := s.pick(title, "*.txt;*.csv")
	if e != nil || path == "" {
		return topics.Set{}, e
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".csv" {
		return topics.Set{}, errors.New("invalidTopics")
	}
	in, e := os.Open(path)
	if e != nil {
		return topics.Set{}, e
	}
	defer func() { _ = in.Close() }()
	data, e := io.ReadAll(io.LimitReader(in, 2*1024*1024+1))
	if e != nil {
		return topics.Set{}, e
	}
	return s.store.Import(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), data, ext == ".csv")
}
