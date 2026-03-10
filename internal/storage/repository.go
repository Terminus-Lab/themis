package storage

import (
	"context"

	"github.com/Terminus-Lab/themis/internal/models"
)

type Repository interface {
	Store(ctx context.Context, evaluation *Evaluation) error
	Query(ctx context.Context, filters models.QueryFilters) ([]Evaluation, int, error)
}
