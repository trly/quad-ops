package cmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/service"
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
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
				gitSyncCalled = true
				return []repository.SyncResult{}, nil
			},
		},
		ComposeProcessor: &MockComposeProcessor{},
		Renderer:         &MockRenderer{},
		ArtifactStore:    &MockArtifactStore{},
		Lifecycle:        &MockLifecycle{},
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
		DryRun: true,
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

// TestSyncCommand_ProcessesComposeProjects verifies compose processor is invoked.
// This is a regression test for GitHub issue #47 where v0.21.0 created a processor
// but never called Process(), resulting in no actual processing.
func TestSyncCommand_ProcessesComposeProjects(t *testing.T) {
	processCalls := 0
	var processedProjects []string

	deps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ fs.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: &MockGitSyncer{
			SyncAllFunc: func(_ context.Context, _ []config.Repository) ([]repository.SyncResult, error) {
				return []repository.SyncResult{
					{Repository: config.Repository{Name: "test-repo"}, Success: true, Changed: true},
				}, nil
			},
		},
		ComposeProcessor: &MockComposeProcessor{
			ProcessFunc: func(_ context.Context, project *types.Project) ([]service.Spec, error) {
				processCalls++
				processedProjects = append(processedProjects, project.Name)
				return []service.Spec{}, nil
			},
		},
		Renderer: &MockRenderer{
			RenderFunc: func(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
				return &platform.RenderResult{Artifacts: []platform.Artifact{}, ServiceChanges: map[string]platform.ChangeStatus{}}, nil
			},
		},
		ArtifactStore: &MockArtifactStore{
			WriteFunc: func(_ context.Context, _ []platform.Artifact) ([]string, error) {
				return []string{}, nil
			},
		},
		Lifecycle: &MockLifecycle{
			ReloadFunc: func(_ context.Context) error { return nil },
		},
	}

	tmpDir := t.TempDir()

	// Create a minimal compose file in the repository directory
	repoDir := filepath.Join(tmpDir, "test-repo")
	err := os.MkdirAll(repoDir, 0750)
	require.NoError(t, err)

	composeContent := []byte(`services:
  web:
    image: nginx:latest
`)
	err = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), composeContent, 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			RepositoryDir: tmpDir,
			Repositories: []config.Repository{
				{Name: "test-repo"},
			},
		}).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	syncCmd := NewSyncCommand()
	opts := SyncOptions{}

	runErr := syncCmd.Run(context.Background(), app, opts, deps)
	require.NoError(t, runErr)
	assert.Greater(t, processCalls, 0, "ComposeProcessor.Process should be called at least once")
}
