package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestShowCommand_Success(t *testing.T) {
	unitManager := &MockUnitManager{
		ShowFunc: func(_, _ string) error {
			return nil
		},
	}

	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"test-unit"})
	assert.NoError(t, err)
	assert.Len(t, unitManager.ShowCalls, 1)
	assert.Equal(t, "test-unit", unitManager.ShowCalls[0].Name)
	assert.Equal(t, "container", unitManager.ShowCalls[0].UnitType)
}

func TestShowCommand_WithCustomType(t *testing.T) {
	unitManager := &MockUnitManager{
		ShowFunc: func(_, _ string) error {
			return nil
		},
	}

	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"--type", "volume", "test-unit"})
	assert.NoError(t, err)
	assert.Len(t, unitManager.ShowCalls, 1)
	assert.Equal(t, "test-unit", unitManager.ShowCalls[0].Name)
	assert.Equal(t, "volume", unitManager.ShowCalls[0].UnitType)
}

func TestShowCommand_InvalidUnitName(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"invalid|unit"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid unit name")
}

func TestShowCommand_UnitManagerError(t *testing.T) {
	unitManager := &MockUnitManager{
		ShowFunc: func(_, _ string) error {
			return errors.New("unit not found")
		},
	}

	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		Build(t)

	cmd := NewShowCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"missing-unit"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to show unit")
	assert.Contains(t, err.Error(), "unit not found")
}

func TestShowCommand_Help(t *testing.T) {
	cmd := NewShowCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Show the contents of a quadlet unit")
	assert.Contains(t, output, "--type")
}

func TestShowCommand_Run(t *testing.T) {
	unitManager := &MockUnitManager{
		ShowFunc: func(_, _ string) error {
			return nil
		},
	}

	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		Build(t)

	showCommand := NewShowCommand()
	opts := ShowOptions{UnitType: "container"}
	deps := ShowDeps{CommonDeps: NewCommonDeps(app.Logger)}

	err := showCommand.Run(context.Background(), app, opts, deps, "test-unit")
	assert.NoError(t, err)
	assert.Len(t, unitManager.ShowCalls, 1)
	assert.Equal(t, "test-unit", unitManager.ShowCalls[0].Name)
	assert.Equal(t, "container", unitManager.ShowCalls[0].UnitType)
}
