package storage

import (
	"context"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

type DB interface {
	Close() error
}

type Repository interface {
	Store(ctx context.Context, evaluation *Evaluation) error
	Query(ctx context.Context, filters models.QueryFilters) ([]Evaluation, int, error)
	QueryById(ctx context.Context, eventID string) (*Evaluation, error)

	GetConversation(ctx context.Context, conversationID string) ([]Evaluation, error)
	ListConversations(ctx context.Context) ([]ConversationSummary, error)

	Sample(ctx context.Context, filters SampleFilters) ([]Evaluation, error)
	SampleConversations(ctx context.Context, filters SampleFilters) ([]ConversationSample, error)
	HealthMetrics(ctx context.Context, since time.Time) (HealthMetricsData, error)
}
