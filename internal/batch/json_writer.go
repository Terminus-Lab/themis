package batch

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type JSONLWriter struct {
	output io.Writer
	logger *zerolog.Logger
}

func NewJSONLWriter(output io.Writer, logger *zerolog.Logger) *JSONLWriter {
	return &JSONLWriter{
		output: output,
		logger: logger,
	}
}

func (w *JSONLWriter) Write(result models.EvaluationResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("Failed to marshal the result. Error: %w", err)
	}

	_, err = w.output.Write(append(data, '\n'))
	return err
}

func (w *JSONLWriter) Close() error {
	return nil
}
