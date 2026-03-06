package batch

import (
	"context"
	"sync"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type Executor interface {
	Execute(ctx context.Context, evalCtx models.EvaluationContext) models.EvaluationResult
}

type Processor struct {
	executor Executor
	workers  int
	logger   *zerolog.Logger
}

func NewProcessor(exec Executor, workers int, logger *zerolog.Logger) *Processor {
	return &Processor{
		executor: exec,
		workers:  workers,
		logger:   logger,
	}
}

// Process takes input records and returns evaluation results via channel
func (p *Processor) Process(ctx context.Context, records []InputRecord) <-chan models.EvaluationResult {
	results := make(chan models.EvaluationResult, len(records))
	jobs := make(chan InputRecord, len(records))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go p.worker(ctx, i, jobs, results, &wg)
	}

	// Send jobs
	p.logger.Info().
		Int("workers", p.workers).
		Int("total_records", len(records)).
		Msg("Starting worker pool")

	for _, record := range records {
		jobs <- record
	}
	close(jobs)

	// Wait and close results channel
	go func() {
		wg.Wait()
		close(results)
		p.logger.Info().Msg("Worker pool finished")
	}()

	return results
}

func (p *Processor) worker(ctx context.Context, workerID int, jobs <-chan InputRecord, results chan<- models.EvaluationResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range jobs {
		if record.Error != nil {
			p.logger.Warn().
				Int("worker", workerID).
				Int("line", record.LineNumber).
				Err(record.Error).
				Msg("Skipping record with parse error")
			continue
		}

		evalCtx := models.EvaluationContext{
			RequestID:      record.Request.EventID,
			Query:          record.Request.Interaction.UserQuery,
			Context:        record.Request.Interaction.Context,
			Answer:         record.Request.Interaction.Answer,
			ExpectedOutput: record.Request.Interaction.ExpectedOutput,
			CreatedAt:      time.Now(),
		}

		result := p.executor.Execute(ctx, evalCtx)
		results <- result
	}

	p.logger.Debug().Int("worker", workerID).Msg("Worker finished")
}
