package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
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

func (e *EvalRepository) Store(ctx context.Context, evaluation *storage.Evaluation) error {
	evalQuery := `
		INSERT INTO eval_results (event_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	stageJsonScore, err := json.Marshal(evaluation.StageScores)
	if err != nil {
		return fmt.Errorf("unable to marshal stage scores. Error: %w", err)
	}

	_, err = e.db.Pool.Exec(
		ctx,
		evalQuery,
		evaluation.EventID,
		evaluation.AgentName,
		evaluation.AgentVersion,
		evaluation.UserQuery,
		evaluation.Answer,
		evaluation.Context,
		evaluation.Confidence,
		evaluation.Verdict,
		stageJsonScore,
	)
	if err != nil {
		return fmt.Errorf("failed to insert evaluation result. Error: %w", err)
	}

	e.logger.
		Info().
		Str("event_id", evaluation.EventID).
		Str("agent_name", evaluation.AgentName).
		Msg("Successfully written to database")

	return nil
}

func (e *EvalRepository) Query(ctx context.Context, filters models.QueryFilters) ([]storage.Evaluation, int, error) {
	// Build WHERE clause and args
	whereClause := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filters.AgentName != "" {
		whereClause += fmt.Sprintf(" AND agent_name = $%d", argIdx)
		args = append(args, filters.AgentName)
		argIdx++
	}

	if filters.Verdict != "" {
		whereClause += fmt.Sprintf(" AND verdict = $%d", argIdx)
		args = append(args, filters.Verdict)
		argIdx++
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM eval_results %s", whereClause)
	var totalCount int
	if err := e.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
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

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filters.Limit)
		argIdx++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := e.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer rows.Close()

	evaluations := []storage.Evaluation{}
	var stageScoreJSON []byte

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

		if err := json.Unmarshal(stageScoreJSON, &evaluation.StageScores); err != nil {
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

func (e *EvalRepository) QueryById(ctx context.Context, eventID string) (*storage.Evaluation, error) {
	query := `
		SELECT event_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
		FROM eval_results
		WHERE event_id = $1
	`

	var evaluation storage.Evaluation
	var stageScoreJSON []byte

	err := e.db.Pool.QueryRow(ctx, query, eventID).Scan(
		&evaluation.EventID,
		&evaluation.AgentName,
		&evaluation.AgentVersion,
		&evaluation.UserQuery,
		&evaluation.Answer,
		&evaluation.Context,
		&evaluation.Confidence,
		&evaluation.Verdict,
		&stageScoreJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query by id. Error: %w", err)
	}

	if err := json.Unmarshal(stageScoreJSON, &evaluation.StageScores); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stage_scores. Error: %w", err)
	}

	return &evaluation, nil
}
