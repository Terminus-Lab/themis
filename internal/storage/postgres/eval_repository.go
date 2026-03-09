package postgres

import (
	"context"
	"encoding/json"
	"fmt"

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
		return fmt.Errorf("Unable to marshal stage scores. Error: %w", err)
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
		Msg("Successfully written in the database")

	return nil
}

func (e *EvalRepository) Query(ctx context.Context, filters storage.QueryFilters) ([]storage.Evaluation, int, error) {
	return nil, 0, nil
}
