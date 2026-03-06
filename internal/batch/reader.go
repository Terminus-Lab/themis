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

type Reader struct {
	file   io.Reader
	logger *zerolog.Logger
}

func NewReader(file io.Reader, logger *zerolog.Logger) *Reader {
	return &Reader{
		file:   file,
		logger: logger,
	}
}

func (r *Reader) ReadAll(ctx context.Context) <-chan InputRecord {
	ch := make(chan InputRecord)

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

			var req models.EvaluationRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				// Send error to the channel
				ch <- InputRecord{LineNumber: lineNum, Error: fmt.Errorf("parse error: %w", err)}
				continue
			}

			// Send success record
			ch <- InputRecord{LineNumber: lineNum, Request: req}
		}

		if err := scanner.Err(); err != nil {
			r.logger.Error().Err(err).Msg("Scanner Error")
		}
	}()

	return ch
}
