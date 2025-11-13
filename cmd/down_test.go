package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/platform"
)

// TestDownCommand_ValidationFailure verifies that validation failures are handled correctly.
func TestDownCommand_ValidationFailure(t *testing.T) {
	// Create app with failing validator
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	// Setup command with app in context
	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute PreRunE (which should trigger validation)
	err := cmd.PreRunE(cmd, []string{})

	// Verify error was returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestDownCommand_StopUnitsSuccess verifies successful unit stopping.
func TestDownCommand_StopUnitsSuccess(t *testing.T) {
	// Mock artifact store to return services
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
				{Path: "/path/to/api.container"},
			}, nil
		},
	}

	// Mock lifecycle for stopping services
	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{
				"web": nil,
				"api": nil,
			}
		},
	}

	// Create app with mocked dependencies including platform components
	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		WithVerbose(true).
		Build(t)

	// Setup command and execute
	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute command using helper
	err := ExecuteCommand(t, cmd, []string{})

	// Verify success
	require.NoError(t, err)
}

// TestDownCommand_WithOutput demonstrates proper output capture using helpers.
func TestDownCommand_WithOutput(t *testing.T) {
	// Mock artifact store to return services
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
			}, nil
		},
	}

	// Mock lifecycle for stopping services
	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{
				"web": nil,
			}
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		WithVerbose(true).
		Build(t)

	// Create command and setup context
	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute with full output capture
	_, err := ExecuteCommandWithCapture(t, cmd, []string{})

	// Verify success
	require.NoError(t, err)
}

// TestDownCommand_MultipleServices verifies stopping multiple specified services.
func TestDownCommand_MultipleServices(t *testing.T) {
	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, services []string) map[string]error {
			result := make(map[string]error)
			for _, svc := range services {
				result[svc] = nil
			}
			return result
		},
	}

	app := NewAppBuilder(t).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--services", "web,api,db"})

	require.NoError(t, err)
}

// TestDownCommand_StopErrors verifies error handling when services fail to stop.
func TestDownCommand_StopErrors(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
				{Path: "/path/to/api.container"},
			}, nil
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{
				"web": nil,
				"api": errors.New("failed to stop api service"),
			}
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop 1 of 2 services")
}

// TestDownCommand_NoServicesFound verifies behavior when no services are found.
func TestDownCommand_NoServicesFound(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{}, nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})

	require.NoError(t, err)
}

// TestDownCommand_ArtifactStoreListError verifies error handling when artifact listing fails.
func TestDownCommand_ArtifactStoreListError(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return nil, errors.New("failed to access artifact store")
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithRenderer(&MockRenderer{}).
		WithLifecycle(&MockLifecycle{}).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list artifacts")
}

// TestDownCommand_PurgeSuccess verifies artifact purging after stopping services.
func TestDownCommand_PurgeSuccess(t *testing.T) {
	var deletedPaths []string

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
				{Path: "/path/to/web-data.volume"},
			}, nil
		},
		DeleteFunc: func(_ context.Context, paths []string) error {
			deletedPaths = paths
			return nil
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{"web": nil}
		},
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--purge"})

	require.NoError(t, err)
	assert.Len(t, deletedPaths, 2)
}

// TestDownCommand_PurgeSpecificServices verifies purging artifacts for specific services only.
func TestDownCommand_PurgeSpecificServices(t *testing.T) {
	var deletedPaths []string

	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				// Artifact paths use the actual unit names (base name without extension)
				{Path: "/path/to/web.container"},
				{Path: "/path/to/api.container"},
			}, nil
		},
		DeleteFunc: func(_ context.Context, paths []string) error {
			deletedPaths = paths
			return nil
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{"web": nil}
		},
		ReloadFunc: func(_ context.Context) error {
			return nil
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--services", "web", "--purge"})

	require.NoError(t, err)
	assert.Len(t, deletedPaths, 1)
	assert.Contains(t, deletedPaths[0], "web")
}

// TestDownCommand_PurgeDeleteError verifies error handling when artifact deletion fails.
func TestDownCommand_PurgeDeleteError(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
			}, nil
		},
		DeleteFunc: func(_ context.Context, _ []string) error {
			return errors.New("permission denied")
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{"web": nil}
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--purge"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete artifacts")
}

// TestDownCommand_PurgeReloadError verifies error handling when service manager reload fails.
func TestDownCommand_PurgeReloadError(t *testing.T) {
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			return []platform.Artifact{
				{Path: "/path/to/web.container"},
			}, nil
		},
		DeleteFunc: func(_ context.Context, _ []string) error {
			return nil
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{"web": nil}
		},
		ReloadFunc: func(_ context.Context) error {
			return errors.New("systemd reload failed")
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--purge"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reload service manager")
}

// TestDownCommand_PurgeListError verifies error handling when artifact listing fails during purge.
func TestDownCommand_PurgeListError(t *testing.T) {
	callCount := 0
	artifactStore := &MockArtifactStore{
		ListFunc: func(_ context.Context) ([]platform.Artifact, error) {
			callCount++
			if callCount == 1 {
				return []platform.Artifact{
					{Path: "/path/to/web.container"},
				}, nil
			}
			return nil, errors.New("failed to list artifacts for purge")
		},
	}

	lifecycle := &MockLifecycle{
		StopManyFunc: func(_ context.Context, _ []string) map[string]error {
			return map[string]error{"web": nil}
		},
	}

	app := NewAppBuilder(t).
		WithArtifactStore(artifactStore).
		WithLifecycle(lifecycle).
		Build(t)

	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--purge"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list artifacts for purge")
}
