package batch

import (
	"fmt"
	"io"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type Writer interface {
	Write(result models.EvaluationResult) error
	Close() error
}

func NewWriter(output io.Writer, format string, logger *zerolog.Logger) (Writer, error) {
	switch format {
	case "jsonl":
		return NewJSONLWriter(output, logger), nil
	case "summary":
		return NewSummaryWriter(output, logger), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
