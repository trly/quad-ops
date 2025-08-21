// Package log provides logging functionality for quad-ops.
package log

import (
	"io"
	"log/slog"
	"os"
)

// Logger defines the interface for logging operations.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogAdapter wraps slog.Logger to implement our Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// Debug logs a debug message.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info logs an info message.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error logs an error message.
func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// NewLogger creates a new logger with the specified verbosity.
func NewLogger(verbose bool) Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}

	if verbose {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	slogLogger := slog.New(handler)

	return &SlogAdapter{logger: slogLogger}
}

// Nop returns a logger that discards all output.
func Nop() Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	slogLogger := slog.New(handler)
	return &SlogAdapter{logger: slogLogger}
}

// NewSlogAdapter creates a Logger from an slog.Logger.
func NewSlogAdapter(slogLogger *slog.Logger) Logger {
	return &SlogAdapter{logger: slogLogger}
}
