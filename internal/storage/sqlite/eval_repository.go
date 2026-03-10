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
	query := `
			SELECT id, event_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores, created_at
			FROM eval_results
			WHERE 1=1
	`

	args := []any{}

	if queryFilters.AgentName != "" {
		query += " AND agent_name = ?"
		args = append(args, queryFilters.AgentName)
	}

	if queryFilters.Verdict != "" {
		query += " AND verdict = ?"
		args = append(args, queryFilters.Verdict)
	}

	query += " ORDER BY created_at DESC"

	if queryFilters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, queryFilters.Limit)
	}

	if queryFilters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, queryFilters.Offset)
	}

	evaluations := []storage.Evaluation{}

	rows, err := e.db.client.QueryContext(ctx, query, args...)
	if err != nil {
		return evaluations, len(evaluations), fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer rows.Close() 

	var (
		id             string
		createdAt      string
		stageScoreJSON string
	)

	for rows.Next() {
		var evaluation storage.Evaluation
		if err := rows.Scan(
			&id,
			&evaluation.EventID,
			&evaluation.AgentName,
			&evaluation.AgentVersion,
			&evaluation.UserQuery,
			&evaluation.Answer,
			&evaluation.Context,
			&evaluation.Confidence,
			&evaluation.Verdict,
			&stageScoreJSON,
			&createdAt,
		); err != nil {
			return nil, len(evaluations), fmt.Errorf("failed to deserialize row. Error: %w", err)
		}

		if err := json.Unmarshal([]byte(stageScoreJSON), &evaluation.StageScores); err != nil {
			return evaluations, len(evaluations), fmt.Errorf("unable to query storage. Error: %w", err)
		}

		evaluations = append(evaluations, evaluation)
	}

	return evaluations, len(evaluations), nil
}
