package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		INSERT INTO eval_results (event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`
	stageJsonScore, err := json.Marshal(evaluation.StageScores)
	if err != nil {
		return fmt.Errorf("unable to marshal stage scores. Error: %w", err)
	}

	_, err = e.db.Pool.Exec(
		ctx,
		evalQuery,
		evaluation.EventID,
		evaluation.ConversationID,
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
		SELECT event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
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
			&evaluation.ConversationID,
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
		SELECT event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
		FROM eval_results
		WHERE event_id = $1
	`

	var evaluation storage.Evaluation
	var stageScoreJSON []byte

	err := e.db.Pool.QueryRow(ctx, query, eventID).Scan(
		&evaluation.EventID,
		&evaluation.ConversationID,
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

func (e *EvalRepository) GetConversation(ctx context.Context, conversationID string) ([]storage.Evaluation, error) {
	var evaluations []storage.Evaluation
	var stageScoreJSON []byte

	if conversationID == "" {
		return evaluations, nil
	}

	query := `
		SELECT event_id, conversation_id, agent_name, agent_version,
			user_query, answer, context, confidence, verdict,
			stage_scores
		FROM eval_results
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`

	rows, err := e.db.Pool.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var evaluation storage.Evaluation
		if err := rows.Scan(
			&evaluation.EventID,
			&evaluation.ConversationID,
			&evaluation.AgentName,
			&evaluation.AgentVersion,
			&evaluation.UserQuery,
			&evaluation.Answer,
			&evaluation.Context,
			&evaluation.Confidence,
			&evaluation.Verdict,
			&stageScoreJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row. Error: %w", err)
		}

		if err := json.Unmarshal(stageScoreJSON, &evaluation.StageScores); err != nil {
			return nil, fmt.Errorf("failed to unmarshal stage_scores. Error: %w", err)
		}

		evaluations = append(evaluations, evaluation)
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return evaluations, nil
}

func (e *EvalRepository) Sample(ctx context.Context, filters storage.SampleFilters) ([]storage.Evaluation, error) {
	// Count total records in date range
	var total int
	if err := e.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM eval_results WHERE created_at >= $1 AND created_at <= $2`,
		filters.StartDate, filters.EndDate,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	if total == 0 {
		return []storage.Evaluation{}, nil
	}

	// Compute sample size
	sampleSize := total * filters.Percentage / 100
	if filters.MinSize > 0 && sampleSize < filters.MinSize {
		sampleSize = filters.MinSize
	}
	if filters.MaxSize > 0 && sampleSize > filters.MaxSize {
		sampleSize = filters.MaxSize
	}
	if sampleSize > total {
		sampleSize = total
	}

	query := `
		SELECT event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
		FROM eval_results
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY RANDOM()
		LIMIT $3
	`

	rows, err := e.db.Pool.Query(ctx, query, filters.StartDate, filters.EndDate, sampleSize)
	if err != nil {
		return nil, fmt.Errorf("failed to sample records: %w", err)
	}
	defer rows.Close()

	var evaluations []storage.Evaluation
	var stageScoreJSON []byte

	for rows.Next() {
		var evaluation storage.Evaluation
		if err := rows.Scan(
			&evaluation.EventID,
			&evaluation.ConversationID,
			&evaluation.AgentName,
			&evaluation.AgentVersion,
			&evaluation.UserQuery,
			&evaluation.Answer,
			&evaluation.Context,
			&evaluation.Confidence,
			&evaluation.Verdict,
			&stageScoreJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal(stageScoreJSON, &evaluation.StageScores); err != nil {
			return nil, fmt.Errorf("failed to unmarshal stage_scores: %w", err)
		}

		evaluations = append(evaluations, evaluation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return evaluations, nil
}

func (e *EvalRepository) HealthMetrics(ctx context.Context, since time.Time) (storage.HealthMetricsData, error) {
	row := e.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(AVG(confidence), 0)
		FROM eval_results
		WHERE created_at >= $1`, since)

	var data storage.HealthMetricsData
	if err := row.Scan(&data.TotalEvaluations, &data.AvgConfidence); err != nil {
		return data, fmt.Errorf("failed to query health metrics: %w", err)
	}

	return data, nil
}

func (e *EvalRepository) ListConversations(ctx context.Context) ([]storage.ConversationSummary, error) {
	query := `
          SELECT
              conversation_id,
              COUNT(*) as turn_count,
              AVG(confidence) as avg_confidence,
              SUM(CASE WHEN verdict = 'pass' THEN 1 ELSE 0 END) as pass_count,
              SUM(CASE WHEN verdict = 'review' THEN 1 ELSE 0 END) as review_count,
              SUM(CASE WHEN verdict = 'fail' THEN 1 ELSE 0 END) as fail_count,
              MIN(created_at) as first_turn_at,
              MAX(created_at) as last_turn_at,
              agent_name,
              agent_version
          FROM eval_results
          WHERE conversation_id IS NOT NULL AND conversation_id != ''
          GROUP BY conversation_id, agent_name, agent_version
          ORDER BY last_turn_at DESC
      `
	rows, err := e.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer rows.Close()

	var summaries []storage.ConversationSummary

	for rows.Next() {
		var summary storage.ConversationSummary
		if err := rows.Scan(
			&summary.ConversationID,
			&summary.TurnCount,
			&summary.AvgConfidence,
			&summary.PassCount,
			&summary.ReviewCount,
			&summary.FailCount,
			&summary.FirstTurnAt,
			&summary.LastTurnAt,
			&summary.AgentName,
			&summary.AgentVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return summaries, nil
}
