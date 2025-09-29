package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/repository"
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
	// Create mocks
	unitManager := &MockUnitManager{}
	unitRepo := &MockUnitRepo{
		FindAllFunc: func() ([]repository.Unit, error) {
			return []repository.Unit{
				{Name: "web"},
				{Name: "api"},
			}, nil
		},
	}

	// Create app with mocked dependencies
	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		WithUnitRepo(unitRepo).
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

	// Verify unit operations were called correctly
	assert.Len(t, unitManager.StopCalls, 2)

	// Verify Stop was called for each unit
	assert.Equal(t, "web", unitManager.StopCalls[0].Name)
	assert.Equal(t, "container", unitManager.StopCalls[0].UnitType)
	assert.Equal(t, "api", unitManager.StopCalls[1].Name)
}

// TestDownCommand_WithOutput demonstrates proper output capture using helpers.
func TestDownCommand_WithOutput(t *testing.T) {
	// Create mocks
	unitManager := &MockUnitManager{}
	unitRepo := &MockUnitRepo{
		FindAllFunc: func() ([]repository.Unit, error) {
			return []repository.Unit{{Name: "web"}}, nil
		},
	}

	app := NewAppBuilder(t).
		WithUnitManager(unitManager).
		WithUnitRepo(unitRepo).
		WithVerbose(true).
		Build(t)

	// Create command and setup context
	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute with full output capture
	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	// Verify success
	require.NoError(t, err)

	// Verify service calls
	assert.Len(t, unitManager.StopCalls, 1)
	assert.Equal(t, "web", unitManager.StopCalls[0].Name)

	// Verify output captured correctly
	assert.Contains(t, output, "Stopping 1 units")
	assert.Contains(t, output, "Successfully stopped 1 units")
}
