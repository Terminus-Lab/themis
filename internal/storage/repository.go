package storage

import (
	"context"
	"time"
)

type DB interface {
	Close() error
}

type Repository interface {
	StoreConversation(ctx context.Context, record *ConversationRecord) error
	GetConversation(ctx context.Context, conversationID string) (*ConversationRecord, error)
	ListConversations(ctx context.Context) ([]ConversationSummary, error)
	HealthMetrics(ctx context.Context, since time.Time) (HealthMetricsData, error)
}
