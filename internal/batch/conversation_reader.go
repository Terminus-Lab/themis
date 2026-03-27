package batch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// normalizeHumanScore converts a 1–5 human score to 0.0–1.0.
// If score is nil or already in [0, 1], it is returned unchanged.
func normalizeHumanScore(score *float64) *float64 {
	if score == nil || *score <= 1.0 {
		return score
	}
	normalized := (*score - 1.0) / 4.0
	return &normalized
}

// ConversationReader reads conversation evaluation records from a JSONL file.
type ConversationReader struct {
	file   io.Reader
	logger *zerolog.Logger
}

func NewConversationReader(file io.Reader, logger *zerolog.Logger) *ConversationReader {
	return &ConversationReader{file: file, logger: logger}
}

func (r *ConversationReader) ReadAll(ctx context.Context) <-chan ConversationInputRecord {
	ch := make(chan ConversationInputRecord)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(r.file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++

			select {
			case <-ctx.Done():
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			type rawRecord struct {
				models.ConversationEvaluationRequest
				HumanAnnotation string   `json:"human_annotation"`
				HumanScore      *float64 `json:"human_score"`
				HumanReason     string   `json:"human_reason"`
			}
			var raw rawRecord
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				ch <- ConversationInputRecord{LineNumber: lineNum, Error: fmt.Errorf("parse error: %w", err)}
				continue
			}

			ch <- ConversationInputRecord{
				LineNumber:  lineNum,
				Request:     raw.ConversationEvaluationRequest,
				HumanLabel:  raw.HumanAnnotation,
				HumanScore:  normalizeHumanScore(raw.HumanScore),
				HumanReason: raw.HumanReason,
			}
		}

		if err := scanner.Err(); err != nil {
			r.logger.Error().Err(err).Msg("Scanner Error")
		}
	}()

	return ch
}
