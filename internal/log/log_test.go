package log

import (
	"testing"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger()

	if logger == nil {
		t.Error("GetLogger() returned nil")
	}
}

func TestNewLogger(t *testing.T) {
	// Test non-verbose logger
	logger := NewLogger(false)
	if logger == nil {
		t.Error("Logger should not be nil")
	}

	// Should be able to call all interface methods without panicking
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")

	// Test verbose logger
	verboseLogger := NewLogger(true)
	if verboseLogger == nil {
		t.Error("Verbose logger should not be nil")
	}

	verboseLogger.Debug("test debug verbose")
	verboseLogger.Info("test info verbose")
}
