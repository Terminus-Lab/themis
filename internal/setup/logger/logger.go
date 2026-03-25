package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		Level(lvl).
		With().
		Timestamp().
		Logger()
}
