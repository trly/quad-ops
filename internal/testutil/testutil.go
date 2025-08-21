// Package testutil provides common test utilities and helpers to reduce boilerplate in test files.
package testutil

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

// NewTestLogger creates a logger that writes to t.Logf for testing.
// This ensures test output is properly captured by the test framework.
func NewTestLogger(t testing.TB) log.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	// Create a custom handler that writes to t.Logf
	handler := &testHandler{t: t, opts: opts}
	slogLogger := slog.New(handler)

	return log.NewSlogAdapter(slogLogger)
}

// ConfigOption allows customization of test config settings.
type ConfigOption func(*config.Settings)

// WithRepositoryDir sets a custom repository directory.
func WithRepositoryDir(dir string) ConfigOption {
	return func(cfg *config.Settings) {
		cfg.RepositoryDir = dir
	}
}

// WithVerbose sets verbose logging.
func WithVerbose(verbose bool) ConfigOption {
	return func(cfg *config.Settings) {
		cfg.Verbose = verbose
	}
}

// WithUserMode sets user mode.
func WithUserMode(userMode bool) ConfigOption {
	return func(cfg *config.Settings) {
		cfg.UserMode = userMode
	}
}

// NewMockConfig creates a config provider for testing with optional customizations.
func NewMockConfig(t testing.TB, opts ...ConfigOption) config.Provider {
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	require.NoError(t, err)

	// Cleanup temp directory when test finishes
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	cfg := &config.Settings{
		RepositoryDir: tmpDir,
		Verbose:       true,
	}

	// Apply any custom options
	for _, opt := range opts {
		opt(cfg)
	}

	configProvider := config.NewDefaultConfigProvider()
	configProvider.SetConfig(cfg)
	return configProvider
}

// SetupTempDir creates a temporary directory and returns it along with cleanup function.
func SetupTempDir(t testing.TB) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-*")
	require.NoError(t, err)

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	// Register cleanup with test framework
	t.Cleanup(cleanup)

	return tmpDir, cleanup
}

// testHandler implements slog.Handler to write to testing.TB.
type testHandler struct {
	t    testing.TB
	opts *slog.HandlerOptions
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, record slog.Record) error {
	h.t.Logf("[%s] %s", record.Level.String(), record.Message)
	return nil
}

func (h *testHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h // For simplicity, ignore attributes in tests
}

func (h *testHandler) WithGroup(_ string) slog.Handler {
	return h // For simplicity, ignore groups in tests
}
