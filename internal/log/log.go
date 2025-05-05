// Package log provides logging functionality for quad-ops.
package log

import (
	"log/slog"
	"os"
)

var log *slog.Logger

// Init initializes the application log.
func Init(verbose bool) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if verbose {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	log = slog.New(handler)
	slog.SetDefault(log)
}

// GetLogger returns the configured logger instance.
func GetLogger() *slog.Logger {
	return log
}
