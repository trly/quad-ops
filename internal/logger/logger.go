// Package logger provides logging functionality for quad-ops
package logger

import (
	"log/slog"
	"os"
)

var log *slog.Logger

// Init initializes the application logger.
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
