package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestDefaultGitSyncer_SyncRepo(t *testing.T) {
	tests := []struct {
		name       string
		repo       config.Repository
		wantErr    bool
		skipReason string
	}{
		{
			name: "sync valid repository",
			repo: config.Repository{
				Name: "test-repo",
				URL:  "https://github.com/trly/quad-ops.git",
			},
			wantErr:    false,
			skipReason: "requires network access",
		},
		{
			name: "sync with reference",
			repo: config.Repository{
				Name:      "test-repo-ref",
				URL:       "https://github.com/trly/quad-ops.git",
				Reference: "main",
			},
			wantErr:    false,
			skipReason: "requires network access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Create temp directory for repos
			tempDir := t.TempDir()

			// Create dependencies
			logger := testutil.NewTestLogger(t)
			mockConfig := testutil.NewMockConfig(t)
			mockConfig.GetConfig().RepositoryDir = tempDir

			syncer := NewGitSyncer(mockConfig, logger)

			// Sync repository
			result := syncer.SyncRepo(context.Background(), tt.repo)

			if tt.wantErr {
				require.Error(t, result.Error)
				assert.False(t, result.Success)
			} else {
				require.NoError(t, result.Error)
				assert.True(t, result.Success)
				assert.Equal(t, tt.repo.Name, result.Repository.Name)
			}
		})
	}
}

func TestDefaultGitSyncer_SyncAll(t *testing.T) {
	tests := []struct {
		name       string
		repos      []config.Repository
		wantErr    bool
		skipReason string
	}{
		{
			name: "sync multiple repositories",
			repos: []config.Repository{
				{Name: "repo1", URL: "https://github.com/trly/quad-ops.git"},
				{Name: "repo2", URL: "https://github.com/trly/quad-ops.git"},
			},
			wantErr:    false,
			skipReason: "requires network access",
		},
		{
			name:       "sync empty repository list",
			repos:      []config.Repository{},
			wantErr:    false,
			skipReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Create temp directory for repos
			tempDir := t.TempDir()

			// Create dependencies
			logger := testutil.NewTestLogger(t)
			mockConfig := testutil.NewMockConfig(t)
			mockConfig.GetConfig().RepositoryDir = tempDir

			syncer := NewGitSyncer(mockConfig, logger)

			// Sync all repositories
			results, err := syncer.SyncAll(context.Background(), tt.repos)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, len(tt.repos))

				// Verify each result
				for i, result := range results {
					assert.Equal(t, tt.repos[i].Name, result.Repository.Name)
				}
			}
		})
	}
}

func TestDefaultGitSyncer_SyncRepo_InvalidRepo(t *testing.T) {
	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Sync invalid repository
	repo := config.Repository{
		Name: "invalid",
		URL:  "https://invalid-url-that-does-not-exist.example.com/repo.git",
	}

	result := syncer.SyncRepo(context.Background(), repo)

	// Should fail but not panic
	assert.False(t, result.Success)
	assert.Error(t, result.Error)
	assert.Equal(t, repo.Name, result.Repository.Name)
}

func TestDefaultGitSyncer_ContextCancellation(t *testing.T) {
	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Create cancelled context
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

func TestDefaultGitSyncer_ParallelSync(t *testing.T) {
	t.Skip("requires network access")

	// Create temp directory for repos
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().RepositoryDir = tempDir

	syncer := NewGitSyncer(mockConfig, logger)

	// Create multiple repositories to sync in parallel
	repos := []config.Repository{
		{Name: "repo1", URL: "https://github.com/trly/quad-ops.git"},
		{Name: "repo2", URL: "https://github.com/trly/quad-ops.git"},
		{Name: "repo3", URL: "https://github.com/trly/quad-ops.git"},
	}

	// Sync all repositories
	results, err := syncer.SyncAll(context.Background(), repos)

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// All should succeed
	for i, result := range results {
		assert.True(t, result.Success, "repo %d should succeed", i)
		assert.NoError(t, result.Error, "repo %d should not error", i)
		assert.Equal(t, repos[i].Name, result.Repository.Name)
	}
}

func TestSyncResult_Structure(t *testing.T) {
	// Test that SyncResult has expected fields
	result := SyncResult{
		Repository: config.Repository{Name: "test", URL: "https://example.com"},
		Success:    true,
		Error:      nil,
		Changed:    true,
		CommitHash: "abc123",
	}

	assert.Equal(t, "test", result.Repository.Name)
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.True(t, result.Changed)
	assert.Equal(t, "abc123", result.CommitHash)
}
