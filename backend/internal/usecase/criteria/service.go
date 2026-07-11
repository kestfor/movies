package criteria

import (
	"context"

	"movies/backend/internal/domain"
)

type Repository interface {
	ListActive(ctx context.Context) ([]domain.Criterion, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListActive(ctx context.Context) ([]domain.Criterion, error) {
	return s.repo.ListActive(ctx)
}
