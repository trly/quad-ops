package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestDefaultGitSyncer_SyncAll(t *testing.T) {
	t.Run("sync empty repository list", func(t *testing.T) {
		// Create temp directory for repos
		tempDir := t.TempDir()

		// Create dependencies
		logger := testutil.NewTestLogger(t)
		mockConfig := testutil.NewMockConfig(t)
		mockConfig.GetConfig().RepositoryDir = tempDir

		syncer := NewGitSyncer(mockConfig, logger)

		// Sync all repositories
		results, err := syncer.SyncAll(context.Background(), []config.Repository{})

		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestDefaultGitSyncer_SyncRepo_InvalidRepo(t *testing.T) {
	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Invalid repository URL
	repo := config.Repository{
		Name: "invalid",
		URL:  "not-a-valid-url",
	}

	result := syncer.SyncRepo(context.Background(), repo)

	assert.Error(t, result.Error)
	assert.False(t, result.Success)
}

func TestDefaultGitSyncer_SyncRepo_EmptyURL(t *testing.T) {
	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Empty URL
	repo := config.Repository{
		Name: "empty-url",
		URL:  "",
	}

	result := syncer.SyncRepo(context.Background(), repo)

	assert.Error(t, result.Error)
	assert.False(t, result.Success)
}

func TestDefaultGitSyncer_ContextCancellation(t *testing.T) {
	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Sync should respect context
	repo := config.Repository{
		Name: "test",
		URL:  "https://github.com/trly/quad-ops.git",
	}

	result := syncer.SyncRepo(ctx, repo)
	assert.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, context.Canceled)

	// SyncAll should respect context
	results, err := syncer.SyncAll(ctx, []config.Repository{repo})
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotNil(t, results)
}
