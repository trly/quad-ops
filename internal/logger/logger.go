package logger

import (
	"log/slog"
	"os"
)

var log *slog.Logger

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

func GetLogger() *slog.Logger {
	return log
}
