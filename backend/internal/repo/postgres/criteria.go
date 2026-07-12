package postgres

import (
	"context"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
)

type CriteriaRepository struct {
	queries *gen.Queries
}

func NewCriteriaRepository(queries *gen.Queries) *CriteriaRepository {
	return &CriteriaRepository{queries: queries}
}

func (r *CriteriaRepository) ListActive(ctx context.Context) ([]domain.Criterion, error) {
	items, err := r.queries.ListActiveCriteria(ctx)
	if err != nil {
		return nil, err
	}

	criteria := make([]domain.Criterion, 0, len(items))
	for _, item := range items {
		criteria = append(criteria, domain.Criterion{
			ID:          item.ID,
			Code:        item.Code,
			Name:        item.Name,
			Description: item.Description,
			SortOrder:   item.SortOrder,
		})
	}
	return criteria, nil
}
