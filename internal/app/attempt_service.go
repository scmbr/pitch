package app

import (
	"github.com/scmbr/pitch/internal/attempts"
	"github.com/scmbr/pitch/internal/settings"
)

type AttemptService struct {
	store    *attempts.Store
	settings *settings.Store
}

func NewAttemptService(store *attempts.Store, preferences *settings.Store) *AttemptService {
	return &AttemptService{store: store, settings: preferences}
}

func (s *AttemptService) List() ([]attempts.Attempt, error) { return s.store.List() }
func (s *AttemptService) Update(id int64, scores attempts.Scores, note string) error {
	return s.store.Update(id, scores, note)
}
func (s *AttemptService) Delete(id int64) error { return s.store.Delete(id) }

type StoragePolicy struct {
	LimitMB    int64 `json:"limitMB"`
	AutoDelete bool  `json:"autoDelete"`
}

func (s *AttemptService) GetPolicy() (StoragePolicy, error) {
	p, err := s.settings.Load()
	return StoragePolicy{LimitMB: p.StorageLimitMB, AutoDelete: p.AutoDelete}, err
}
func (s *AttemptService) SetPolicy(policy StoragePolicy) error {
	p, err := s.settings.Load()
	if err != nil {
		return err
	}
	p.StorageLimitMB, p.AutoDelete = policy.LimitMB, policy.AutoDelete
	return s.settings.Save(p)
}
