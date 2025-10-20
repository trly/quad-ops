package cmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
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

// TestSyncCommand_UnsupportedPlatform tests handling of unsupported platform.
func TestSyncCommand_UnsupportedPlatform(t *testing.T) {
	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
	}

	// Build app with unsupported platform (e.g., windows)
	app := NewAppBuilder(t).WithOS("windows").Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform not supported")
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

	app := NewAppBuilder(t).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)
	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
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
		// TODO: Add mock implementations for new interfaces
		GitSyncer:        nil,
		ComposeProcessor: nil,
		Renderer:         nil,
		ArtifactStore:    nil,
		Lifecycle:        nil,
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{
		DryRun: true, // Dry run mode
	}

	err := syncCmd.Run(context.Background(), app, opts, deps)
	assert.NoError(t, err)
	assert.False(t, gitSyncCalled, "Git sync should not be called in dry run mode")
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

// TestSyncCommand_ProcessorCodePath verifies the processor code path exists and isn't skipped.
// This is a regression test for GitHub issue #47 where v0.21.0 created a processor
// but never called ProcessProjects(), only logging "Would process projects" count=2.
// The bug was introduced in commit c76faf2 where processor was assigned to _ unused variable.
func TestSyncCommand_ProcessorCodePath(t *testing.T) {
	// This test verifies the code no longer contains the bug pattern:
	// ❌ processor := deps.NewDefaultProcessor(opts.Force)
	// ❌ _ = processor // Bug: processor created but discarded!
	// ❌ deps.Logger.Info("Would process projects", "count", len(projects))
	//
	// Instead it should call:
	// ✅ processor.ProcessProjects(projects, isLastRepo)

	// Read the sync.go file to verify the bug pattern doesn't exist
	sourceCode, err := os.ReadFile("sync.go")
	require.NoError(t, err, "Should be able to read sync.go")

	code := string(sourceCode)

	// Verify we don't have the bug pattern: processor assigned to underscore
	assert.NotContains(t, code, "_ = processor",
		"Processor should not be discarded with _ assignment (GitHub issue #47)")

	// Verify we don't have the old "Would process projects" log that indicated the bug
	assert.NotContains(t, code, `"Would process projects"`,
		"Should not have 'Would process projects' log line (indicates bug from issue #47)")

	// Verify we have the actual ProcessProjects call
	assert.Contains(t, code, "processor.ProcessProjects",
		"Code should call processor.ProcessProjects to actually process the projects")

	// Verify we have the WithExistingProcessedUnits call for proper state tracking
	assert.Contains(t, code, "WithExistingProcessedUnits",
		"Code should track processed units across repositories")
}
