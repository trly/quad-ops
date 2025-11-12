package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
)

func TestNewTestLogger(t *testing.T) {
	logger := NewTestLogger(t)
	assert.NotNil(t, logger)

	// Test that we can call logger methods without panic
	logger.Debug("test debug message")
	logger.Info("test info message")
	logger.Warn("test warn message")
	logger.Error("test error message")
}

func TestNewMockConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		provider := NewMockConfig(t)
		require.NotNil(t, provider)

		cfg := provider.GetConfig()
		require.NotNil(t, cfg)
		assert.True(t, cfg.Verbose)
		assert.NotEmpty(t, cfg.RepositoryDir)

		// Verify temp directory was created
		assert.DirExists(t, cfg.RepositoryDir)
	})

	t.Run("with options", func(t *testing.T) {
		provider := NewMockConfig(t,
			WithRepositoryDir("/custom/path"),
			WithVerbose(false),
			WithUserMode(true))

		cfg := provider.GetConfig()
		assert.Equal(t, "/custom/path", cfg.RepositoryDir)
		assert.False(t, cfg.Verbose)
		assert.True(t, cfg.UserMode)
	})
}

func TestConfigOptions(t *testing.T) {
	t.Run("WithRepositoryDir", func(t *testing.T) {
		cfg := &config.Settings{}
		opt := WithRepositoryDir("/test/path")
		opt(cfg)
		assert.Equal(t, "/test/path", cfg.RepositoryDir)
	})

	t.Run("WithVerbose", func(t *testing.T) {
		cfg := &config.Settings{}
		opt := WithVerbose(true)
		opt(cfg)
		assert.True(t, cfg.Verbose)
	})

	t.Run("WithUserMode", func(t *testing.T) {
		cfg := &config.Settings{}
		opt := WithUserMode(true)
		opt(cfg)
		assert.True(t, cfg.UserMode)
	})
}
