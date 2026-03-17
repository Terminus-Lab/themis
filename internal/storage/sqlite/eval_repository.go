package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		(id, event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	id := uuid.New().String()

	stageScoresJSON, err := json.Marshal(eval.StageScores)
	if err != nil {
		return fmt.Errorf("failed to marshal stage score. Error: %w", err)
	}

	_, err = e.db.client.ExecContext(ctx, query,
		id,
		eval.EventID,
		eval.ConversationID,
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
		SELECT event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
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
	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("Failed to close database rows")
		}
	}()

	evaluations := []storage.Evaluation{}
	var stageScoreJSON string

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

func (e *EvalRepository) QueryById(ctx context.Context, eventID string) (*storage.Evaluation, error) {
	query := `
		SELECT event_id, conversation_id, agent_name, agent_version, user_query, answer, context, confidence, verdict, stage_scores
		FROM eval_results
		WHERE event_id = ?
	`

	var evaluation storage.Evaluation
	var stageScoreJSON string

	err := e.db.client.QueryRowContext(ctx, query, eventID).Scan(
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

	if err := json.Unmarshal([]byte(stageScoreJSON), &evaluation.StageScores); err != nil {
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
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`

	rows, err := e.db.client.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("unable to query storage. Error: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("Failed to close database rows")
		}
	}()

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
	const sqliteFormat = "2006-01-02 15:04:05"
	start := filters.StartDate.UTC().Format(sqliteFormat)
	end := filters.EndDate.UTC().Format(sqliteFormat)

	// Count total records in date range
	countQuery := `SELECT COUNT(*) FROM eval_results WHERE created_at >= ? AND created_at <= ?`
	var total int
	if err := e.db.client.QueryRowContext(ctx, countQuery, start, end).Scan(&total); err != nil {
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
		WHERE created_at >= ? AND created_at <= ?
		ORDER BY RANDOM()
		LIMIT ?
	`

	rows, err := e.db.client.QueryContext(ctx, query, start, end, sampleSize)
	if err != nil {
		return nil, fmt.Errorf("failed to sample records: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("Failed to close database rows")
		}
	}()

	var evaluations []storage.Evaluation
	var stageScoreJSON string

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

		if err := json.Unmarshal([]byte(stageScoreJSON), &evaluation.StageScores); err != nil {
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
	const sqliteFormat = "2006-01-02 15:04:05"
	sinceStr := since.UTC().Format(sqliteFormat)

	row := e.db.client.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(AVG(confidence), 0)
		FROM eval_results
		WHERE created_at >= ?`, sinceStr)

	var data storage.HealthMetricsData
	if err := row.Scan(&data.TotalEvaluations, &data.AvgConfidence); err != nil {
		return data, fmt.Errorf("failed to query health metrics: %w", err)
	}

	if data.TotalEvaluations == 0 {
		return data, nil
	}

	// Compute avg disagreement rate from stage_scores
	rows, err := e.db.client.QueryContext(ctx,
		`SELECT stage_scores FROM eval_results WHERE created_at >= ?`, sinceStr)
	if err != nil {
		return data, fmt.Errorf("failed to query stage scores: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("Failed to close rows")
		}
	}()

	var totalDisagreement float64
	var evalCount int

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return data, fmt.Errorf("failed to scan stage_scores: %w", err)
		}
		var stages []models.StageResult
		if err := json.Unmarshal([]byte(raw), &stages); err != nil {
			continue
		}

		var scores []float64
		for _, s := range stages {
			if s.Score > 0 {
				scores = append(scores, s.Score)
			}
		}

		if len(scores) > 1 {
			totalDisagreement += storage.PopulationStdDev(scores)
			evalCount++
		}
	}
	if err := rows.Err(); err != nil {
		return data, fmt.Errorf("error iterating stage_scores: %w", err)
	}

	if evalCount > 0 {
		data.AvgDisagreementRate = totalDisagreement / float64(evalCount)
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
	rows, err := e.db.client.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unable to query conversations: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			e.logger.Error().Err(err).Msg("Failed to close database rows")
		}
	}()

	var summaries []storage.ConversationSummary
	// format for sqlite
	const sqliteDateTime = "2006-01-02 15:04:05"

	for rows.Next() {
		var summary storage.ConversationSummary
		var firstTurnAt, lastTurnAt string
		if err := rows.Scan(
			&summary.ConversationID,
			&summary.TurnCount,
			&summary.AvgConfidence,
			&summary.PassCount,
			&summary.ReviewCount,
			&summary.FailCount,
			&firstTurnAt,
			&lastTurnAt,
			&summary.AgentName,
			&summary.AgentVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if t, err := time.Parse(sqliteDateTime, firstTurnAt); err == nil {
			summary.FirstTurnAt = t
		}
		if t, err := time.Parse(sqliteDateTime, lastTurnAt); err == nil {
			summary.LastTurnAt = t
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return summaries, nil
}
