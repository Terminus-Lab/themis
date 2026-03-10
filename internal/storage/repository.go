package storage

import (
	"context"

	"github.com/Terminus-Lab/themis/internal/models"
)

type DB interface {
	Close() error
}

type Repository interface {
	Store(ctx context.Context, evaluation *Evaluation) error
	Query(ctx context.Context, filters models.QueryFilters) ([]Evaluation, int, error)
	QueryById(ctx context.Context, eventID string) (*Evaluation, error)
}
