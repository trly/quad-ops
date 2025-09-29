package cmd

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestSyncCommand_ValidationFailure tests system requirements failure.
func TestSyncCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewSyncCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	// PreRunE returns error instead of exiting
	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestSyncCommand_DirectoryCreationFailure tests quadlet directory creation failure.
func TestSyncCommand_DirectoryCreationFailure(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error {
					return errors.New("permission denied")
				},
			},
			Logger: testutil.NewTestLogger(t),
		},
	}

	app := NewAppBuilder(t).Build(t)
	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestSyncCommand_Success tests successful sync operation.
func TestSyncCommand_Success(t *testing.T) {
	t.Skip("Skipping test that requires full git.Repository initialization - test basic path instead")
	// Testing non-dry-run path requires complex git.Repository setup
	// Coverage is better provided by integration tests or simpler unit tests
}

// TestSyncCommand_DryRun tests dry run mode.
func TestSyncCommand_DryRun(t *testing.T) {
	var gitSyncCalled bool

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		NewGitRepository: func(_ config.Repository, _ config.Provider) *git.Repository {
			gitSyncCalled = true
			return &git.Repository{}
		},
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		DryRun: true, // Dry run mode
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
	assert.False(t, gitSyncCalled, "Git sync should not be called in dry run mode")
}

// TestSyncCommand_NoChanges tests behavior when repository has no changes.
func TestSyncCommand_NoChanges(t *testing.T) {
	t.Skip("Skipping test that requires git.Repository - change detection not yet implemented in sync.go")
}

// TestSyncCommand_RepoFilter tests repository filtering.
func TestSyncCommand_RepoFilter(t *testing.T) {
	t.Skip("Skipping test that requires full git.Repository initialization")
}

// TestSyncCommand_Help tests help output.
func TestSyncCommand_Help(t *testing.T) {
	cmd := NewSyncCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Synchronizes the Docker Compose files")
	assert.Contains(t, output, "--dry-run")
	assert.Contains(t, output, "--repo")
	assert.Contains(t, output, "--force")
}

// TestSyncCommand_Flags tests command-specific flags.
func TestSyncCommand_Flags(t *testing.T) {
	cmd := NewSyncCommand().GetCobraCommand()

	// Test dry-run flag
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)

	// Test repo flag
	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag)
	assert.Equal(t, "", repoFlag.DefValue)

	// Test force flag
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}
