package storage

import (
	"context"

	"github.com/Terminus-Lab/themis/internal/models"
)

type ResultsRepository interface {
	Store(ctx context.Context, result *models.EvaluationResult) error
	Query(ctx context.Context, filters QueryFilters) (*QueryResult, error)
	GetById(ctx context.Context, id string) (*models.EvaluationResult, error)
}
