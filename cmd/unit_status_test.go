package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/platform"
)

func TestStatusCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewStatusCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestStatusCommand_Success(t *testing.T) {
	lifecycle := &MockLifecycle{
		StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
			return &platform.ServiceStatus{
				Name:        "test-service",
				Active:      true,
				State:       "running",
				Description: "Test service",
			}, nil
		},
	}

	app := NewAppBuilder(t).
		WithLifecycle(lifecycle).
		Build(t)

	cmd := NewStatusCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"test-service"})
	assert.NoError(t, err)
}

func TestStatusCommand_InvalidUnitName(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	cmd := NewStatusCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"invalid|unit"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid service name")
}

func TestStatusCommand_LifecycleError(t *testing.T) {
	lifecycle := &MockLifecycle{
		StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
			return nil, errors.New("service not found")
		},
	}

	app := NewAppBuilder(t).
		WithLifecycle(lifecycle).
		Build(t)

	cmd := NewStatusCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"missing-service"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get status")
	assert.Contains(t, err.Error(), "service not found")
}

func TestStatusCommand_Help(t *testing.T) {
	cmd := NewStatusCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Show the status of a service")
}

func TestStatusCommand_Run(t *testing.T) {
	// Mock lifecycle to be returned by GetLifecycle
	lifecycle := &MockLifecycle{
		StatusFunc: func(_ context.Context, _ string) (*platform.ServiceStatus, error) {
			return &platform.ServiceStatus{
				Name:   "test-service",
				Active: true,
				State:  "running",
			}, nil
		},
	}

	app := NewAppBuilder(t).
		WithLifecycle(lifecycle).
		Build(t)

	statusCommand := NewStatusCommand()
	opts := StatusOptions{}
	deps := StatusDeps{CommonDeps: NewCommonDeps(app.Logger)}

	err := statusCommand.Run(context.Background(), app, opts, deps, "test-service")
	assert.NoError(t, err)
}
