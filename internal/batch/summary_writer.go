package batch

import (
	"encoding/json"
	"io"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type SummaryStats struct {
	Total         int     `json:"total"`
	PassCount     int     `json:"pass_count"`
	FailCount     int     `json:"fail_count"`
	ReviewCount   int     `json:"review_count"`
	AvgConfidence float64 `json:"avg_confidence"`
}

type SummaryWriter struct {
	output  io.Writer
	logger  *zerolog.Logger
	results []models.EvaluationResult
}

func NewSummaryWriter(output io.Writer, logger *zerolog.Logger) *SummaryWriter {
	return &SummaryWriter{
		output:  output,
		logger:  logger,
		results: []models.EvaluationResult{},
	}
}

func (w *SummaryWriter) Write(result models.EvaluationResult) error {
	// Collect results
	w.results = append(w.results, result)
	return nil
}

func (w *SummaryWriter) Close() error {
	stats := w.computeStats()

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.output.Write(data)
	return err
}

func (w *SummaryWriter) computeStats() SummaryStats {
	stats := SummaryStats{
		Total: len(w.results),
	}

	var totalConfidence float64

	for _, result := range w.results {
		totalConfidence += result.Confidence

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
		stats.AvgConfidence = totalConfidence / float64(stats.Total)
	}

	return stats
}
