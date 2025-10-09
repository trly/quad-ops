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
				{Path: "/path/to/web-container.container"},
				{Path: "/path/to/api-container.container"},
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
				{Path: "/path/to/web-container.container"},
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
