package storage

import (
	"context"
)

type Repository interface {
	Store(ctx context.Context, evaluation *Evaluation)
	Query(ctx context.Context, filters QueryFilters) ([]Evaluation, int, error)
}
