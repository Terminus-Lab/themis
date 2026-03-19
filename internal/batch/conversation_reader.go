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

			var req models.ConversationEvaluationRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				ch <- ConversationInputRecord{LineNumber: lineNum, Error: fmt.Errorf("parse error: %w", err)}
				continue
			}

			ch <- ConversationInputRecord{LineNumber: lineNum, Request: req}
		}

		if err := scanner.Err(); err != nil {
			r.logger.Error().Err(err).Msg("Scanner Error")
		}
	}()

	return ch
}
