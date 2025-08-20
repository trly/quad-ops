// Package log provides logging functionality for quad-ops.
package log

import (
	"log/slog"
	"os"
)

var log *slog.Logger

// Init initializes the application log.
// Default level is Warn to follow Rule of Silence - only log surprising events.
func Init(verbose bool) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelWarn,
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
