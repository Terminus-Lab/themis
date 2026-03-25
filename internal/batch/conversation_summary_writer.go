package batch

import (
	"encoding/json"
	"io"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// ConversationSummaryStats holds aggregate statistics for a batch of conversation evaluations.
type ConversationSummaryStats struct {
	Total             int                `json:"total"`
	PassCount         int                `json:"pass_count"`
	FailCount         int                `json:"fail_count"`
	ReviewCount       int                `json:"review_count"`
	AvgConfidence     float64            `json:"avg_confidence"`
	AvgTurnCount      float64            `json:"avg_turn_count"`
	CorrelationReport *CorrelationReport `json:"correlation_report,omitempty"`
}

// ConversationSummaryWriter accumulates conversation evaluation results and writes aggregate stats.
type ConversationSummaryWriter struct {
	output            io.Writer
	logger            *zerolog.Logger
	results           []models.ConversationEvaluationResult
	correlationReport *CorrelationReport
}

func NewConversationSummaryWriter(output io.Writer, logger *zerolog.Logger) *ConversationSummaryWriter {
	return &ConversationSummaryWriter{
		output:  output,
		logger:  logger,
		results: []models.ConversationEvaluationResult{},
	}
}

func (w *ConversationSummaryWriter) Write(result models.ConversationEvaluationResult) error {
	w.results = append(w.results, result)
	return nil
}

// SetCorrelationReport attaches a pre-computed correlation report to be included in the summary output.
func (w *ConversationSummaryWriter) SetCorrelationReport(r *CorrelationReport) {
	w.correlationReport = r
}

func (w *ConversationSummaryWriter) Close() error {
	stats := w.computeStats()
	stats.CorrelationReport = w.correlationReport

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.output.Write(data)
	return err
}

func (w *ConversationSummaryWriter) computeStats() ConversationSummaryStats {
	stats := ConversationSummaryStats{
		Total: len(w.results),
	}

	var totalFinalScore float64
	var totalTurns int

	for _, result := range w.results {
		totalFinalScore += result.FinalScore
		totalTurns += result.TurnCount

		switch result.Verdict {
		case models.VerdictPass:
			stats.PassCount++
		case models.VerdictFail:
			stats.FailCount++
		case models.VerdictReview:
			stats.ReviewCount++
		}
	}

	if stats.Total > 0 {
		stats.AvgConfidence = totalFinalScore / float64(stats.Total)
		stats.AvgTurnCount = float64(totalTurns) / float64(stats.Total)
	}

	return stats
}
