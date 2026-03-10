package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Terminus-Lab/themis/internal/models"
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

func (e *EvalRepository) Store(ctx context.Context, eval *storage.Evaluation) error {
	query := `
		INSERT INTO eval_results
		(id, event_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	id := uuid.New().String()

	stageScoresJSON, err := json.Marshal(eval.StageScores)
	if err != nil {
		return fmt.Errorf("failed to marshal stage score. Error: %w", err)
	}

	_, err = e.db.client.ExecContext(ctx, query,
		id,
		eval.EventID,
		eval.AgentName,
		eval.AgentVersion,
		eval.UserQuery,
		eval.Answer,
		eval.Context,
		eval.Confidence,
		eval.Verdict,
		string(stageScoresJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}

	e.logger.Info().
		Str("event_id", eval.EventID).
		Msg("Stored in SQLite")
	return nil
}

func (e *EvalRepository) Query(ctx context.Context, queryFilters models.QueryFilters) ([]storage.Evaluation, int, error) {
	// Build WHERE clause and args
	whereClause := "WHERE 1=1"
	args := []any{}

	if queryFilters.AgentName != "" {
		whereClause += " AND agent_name = ?"
		args = append(args, queryFilters.AgentName)
	}

	if queryFilters.Verdict != "" {
		whereClause += " AND verdict = ?"
		args = append(args, queryFilters.Verdict)
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM eval_results %s", whereClause)
	var totalCount int
	if err := e.db.client.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Short-circuit if no results
	if totalCount == 0 {
		return []storage.Evaluation{}, 0, nil
	}

	// Build data query
	query := fmt.Sprintf(`
		SELECT event_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
		FROM eval_results
		%s
		ORDER BY created_at DESC
	`, whereClause)

	if queryFilters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, queryFilters.Limit)
	}

	if queryFilters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, queryFilters.Offset)
	}

	rows, err := e.db.client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer rows.Close()

	evaluations := []storage.Evaluation{}
	var stageScoreJSON string

	for rows.Next() {
		var evaluation storage.Evaluation
		if err := rows.Scan(
			&evaluation.EventID,
			&evaluation.AgentName,
			&evaluation.AgentVersion,
			&evaluation.UserQuery,
			&evaluation.Answer,
			&evaluation.Context,
			&evaluation.Confidence,
			&evaluation.Verdict,
			&stageScoreJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan row. Error: %w", err)
		}

		if err := json.Unmarshal([]byte(stageScoreJSON), &evaluation.StageScores); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal stage_scores. Error: %w", err)
		}

		evaluations = append(evaluations, evaluation)
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return evaluations, totalCount, nil
}
