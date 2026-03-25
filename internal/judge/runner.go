package judge

import (
	"context"
	"sync"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type JudgeRunner struct {
	Judges []Judge
	logger *zerolog.Logger
}

func NewJudgeRunner(judges []Judge, logger *zerolog.Logger) *JudgeRunner {
	return &JudgeRunner{
		Judges: judges,
		logger: logger,
	}
}

func (c *JudgeRunner) Run(ctx context.Context, evaluationContext models.EvaluationContext) []models.StageResult {
	results := make(chan models.StageResult, len(c.Judges))
	var wg sync.WaitGroup

	judgeTimeout := 15 * time.Second

	for _, judge := range c.Judges {
		wg.Add(1)
		go func(j Judge) {
			defer wg.Done()

			// Create a context with timeout to block the queue
			judgeCtx, cancel := context.WithTimeout(ctx, judgeTimeout)
			defer cancel()

			// run the judge with timeout
			evalResult := j.Evaluate(judgeCtx, evaluationContext)

			// Check if timeout occurred
			if judgeCtx.Err() == context.DeadlineExceeded {
				c.logger.Warn().
					Str("judge_name", evalResult.Name).
					Dur("timeout", judgeTimeout).
					Msg("Judge evaluation timed out")

				// Return a failed result instead of blocking
				evalResult = models.StageResult{
					Name:     evalResult.Name,
					Score:    0.0,
					Error:    "evaluation timed out after " + judgeTimeout.String(),
					Reason:   "evaluation timed out after " + judgeTimeout.String(),
					Duration: judgeTimeout,
					Weight:   evalResult.Weight,
				}
			}

			results <- evalResult
		}(judge)
	}

	wg.Wait()
	close(results)

	var stageResults []models.StageResult
	for result := range results {
		stageResults = append(stageResults, result)
	}
	c.logger.Debug().Int("judgeCount", len(stageResults)).Msg("all judges completed")

	return stageResults
}
