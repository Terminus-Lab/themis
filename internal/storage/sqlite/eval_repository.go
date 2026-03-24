package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type EvalRepository struct {
	db     *DB
	logger *zerolog.Logger
}

func NewEvalRepository(db *DB, logger *zerolog.Logger) *EvalRepository {
	return &EvalRepository{
		db:     db,
		logger: logger,
	}
}

func (e *EvalRepository) StoreConversation(ctx context.Context, record *storage.ConversationRecord) error {
	query := `
		INSERT INTO conversations
		(id, conversation_id, agent_name, agent_version, turn_count, turn_avg, holistic_score,
		 holistic_reason, final_score, verdict, turn_results, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	id := record.ID
	if id == "" {
		id = uuid.New().String()
	}

	turnResultsJSON, err := json.Marshal(record.TurnResults)
	if err != nil {
		return fmt.Errorf("failed to marshal turn_results: %w", err)
	}

	_, err = e.db.client.ExecContext(ctx, query,
		id,
		record.ConversationID,
		record.AgentName,
		record.AgentVersion,
		record.TurnCount,
		record.TurnAvg,
		record.HolisticScore,
		record.HolisticReason,
		record.FinalScore,
		record.Verdict,
		string(turnResultsJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to insert conversation: %w", err)
	}

	e.logger.Info().
		Str("conversation_id", record.ConversationID).
		Msg("stored conversation evaluation in SQLite")
	return nil
}

func (e *EvalRepository) GetConversation(ctx context.Context, conversationID string) (*storage.ConversationRecord, error) {
	query := `
		SELECT id, conversation_id, agent_name, agent_version, turn_count,
		       turn_avg, holistic_score, holistic_reason, final_score, verdict, turn_results, created_at
		FROM conversations
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var record storage.ConversationRecord
	var turnResultsJSON string
	var createdAt string

	err := e.db.client.QueryRowContext(ctx, query, conversationID).Scan(
		&record.ID,
		&record.ConversationID,
		&record.AgentName,
		&record.AgentVersion,
		&record.TurnCount,
		&record.TurnAvg,
		&record.HolisticScore,
		&record.HolisticReason,
		&record.FinalScore,
		&record.Verdict,
		&turnResultsJSON,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if err := json.Unmarshal([]byte(turnResultsJSON), &record.TurnResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal turn_results: %w", err)
	}

	const sqliteDateTime = "2006-01-02 15:04:05"
	if t, err := time.Parse(sqliteDateTime, createdAt); err == nil {
		record.CreatedAt = t
	}

	return &record, nil
}

func (e *EvalRepository) ListConversations(ctx context.Context) ([]storage.ConversationSummary, error) {
	query := `
		SELECT conversation_id, agent_name, agent_version, turn_count, final_score, verdict, created_at
		FROM conversations
		ORDER BY created_at DESC
	`

	rows, err := e.db.client.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("failed to close rows")
		}
	}()

	const sqliteDateTime = "2006-01-02 15:04:05"
	var summaries []storage.ConversationSummary

	for rows.Next() {
		var s storage.ConversationSummary
		var createdAt string
		if err := rows.Scan(
			&s.ConversationID,
			&s.AgentName,
			&s.AgentVersion,
			&s.TurnCount,
			&s.FinalScore,
			&s.Verdict,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if t, err := time.Parse(sqliteDateTime, createdAt); err == nil {
			s.CreatedAt = t
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return summaries, nil
}

func (e *EvalRepository) HealthMetrics(ctx context.Context, since time.Time) (storage.HealthMetricsData, error) {
	const sqliteFormat = "2006-01-02 15:04:05"
	sinceStr := since.UTC().Format(sqliteFormat)

	row := e.db.client.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(AVG(final_score), 0)
		FROM conversations
		WHERE created_at >= ?`, sinceStr)

	var data storage.HealthMetricsData
	if err := row.Scan(&data.TotalEvaluations, &data.AvgConfidence); err != nil {
		return data, fmt.Errorf("failed to query health metrics: %w", err)
	}

	return data, nil
}

// Ensure EvalRepository implements storage.Repository.
var _ storage.Repository = (*EvalRepository)(nil)
