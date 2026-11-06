package app

import (
	"context"
	"github.com/scmbr/pitch/internal/projects"
)

type ProjectService struct {
	store *projects.Store
	pick  func(string, string) (string, error)
}

func NewProjectService(store *projects.Store, pick func(string, string) (string, error)) *ProjectService {
	return &ProjectService{store: store, pick: pick}
}
func (s *ProjectService) List() ([]projects.Project, error)             { return s.store.List() }
func (s *ProjectService) Versions(id int64) ([]projects.Version, error) { return s.store.Versions(id) }
func (s *ProjectService) Import(ctx context.Context, title string) (projects.Version, error) {
	path, e := s.pick(title, "*.pptx;*.pdf")
	if e != nil || path == "" {
		return projects.Version{}, e
	}
	return s.store.Import(ctx, path)
}
func (s *ProjectService) Refresh(ctx context.Context, id int64) (projects.Version, error) {
	items, e := s.store.List()
	if e != nil {
		return projects.Version{}, e
	}
	for _, p := range items {
		if p.ID == id {
			return s.store.Import(ctx, p.Path)
		}
	}
	return projects.Version{}, nil
}
func (s *ProjectService) AttachPDF(id int64, title string) error {
	path, e := s.pick(title, "*.pdf")
	if e != nil || path == "" {
		return e
	}
	return s.store.AttachPDF(id, path)
}
func (s *ProjectService) CacheSlide(id int64, page int, png string) error {
	return s.store.CacheSlide(id, page, png)
}
func (s *ProjectService) CompleteSlides(id int64, count int) error {
	return s.store.CompleteSlides(id, count)
}
