package batch

import (
	"context"
	"sync"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// ConversationExecutor runs conversation-level evaluation.
type ConversationExecutor interface {
	Execute(ctx context.Context, req models.ConversationEvaluationRequest) models.ConversationEvaluationResult
}

// ConversationProcessor processes conversation records concurrently.
type ConversationProcessor struct {
	executor ConversationExecutor
	workers  int
	logger   *zerolog.Logger
}

func NewConversationProcessor(exec ConversationExecutor, workers int, logger *zerolog.Logger) *ConversationProcessor {
	return &ConversationProcessor{
		executor: exec,
		workers:  workers,
		logger:   logger,
	}
}

func (p *ConversationProcessor) Process(ctx context.Context, records []ConversationInputRecord) <-chan models.ConversationEvaluationResult {
	results := make(chan models.ConversationEvaluationResult, len(records))
	jobs := make(chan ConversationInputRecord, len(records))

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go p.worker(ctx, i, jobs, results, &wg)
	}

	p.logger.Info().
		Int("workers", p.workers).
		Int("total_records", len(records)).
		Msg("Starting conversation worker pool")

	for _, record := range records {
		jobs <- record
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
		p.logger.Info().Msg("Conversation worker pool finished")
	}()

	return results
}

func (p *ConversationProcessor) worker(ctx context.Context, workerID int, jobs <-chan ConversationInputRecord, results chan<- models.ConversationEvaluationResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range jobs {
		if record.Error != nil {
			p.logger.Warn().
				Int("worker", workerID).
				Int("line", record.LineNumber).
				Err(record.Error).
				Msg("Skipping conversation record with parse error")
			continue
		}

		result := p.executor.Execute(ctx, record.Request)
		results <- result
	}

	p.logger.Debug().Int("worker", workerID).Msg("Conversation worker finished")
}
