package log

import (
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "default logging level",
			verbose: false,
		},
		{
			name:    "verbose logging level",
			verbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.verbose)
			logger := GetLogger()

			if logger == nil {
				t.Error("expected logger to be initialized, got nil")
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	Init(false)
	logger := GetLogger()

	if logger == nil {
		t.Error("GetLogger() returned nil")
	}

	if logger != log {
		t.Error("GetLogger() returned different logger instance than initialized")
	}
}
