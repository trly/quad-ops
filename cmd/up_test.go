package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/repository"
)

// TestUpCommand_ValidationFailure verifies that validation failures are handled correctly.
func TestUpCommand_ValidationFailure(t *testing.T) {
	// Create app with failing validator
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	// Setup command with app in context
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute PreRunE (which should trigger validation)
	err := cmd.PreRunE(cmd, []string{})

	// Verify error was returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestUpCommand_StartUnitsSuccess verifies successful unit starting.
func TestUpCommand_StartUnitsSuccess(t *testing.T) {
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
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute command using helper
	err := ExecuteCommand(t, cmd, []string{})

	// Verify success
	require.NoError(t, err)

	// Verify unit operations were called correctly
	assert.Len(t, unitManager.ResetFailedCalls, 2)
	assert.Len(t, unitManager.StartCalls, 2)

	// Verify ResetFailed was called for each unit
	assert.Equal(t, "web", unitManager.ResetFailedCalls[0].Name)
	assert.Equal(t, "container", unitManager.ResetFailedCalls[0].UnitType)
	assert.Equal(t, "api", unitManager.ResetFailedCalls[1].Name)

	// Verify Start was called for each unit
	assert.Equal(t, "web", unitManager.StartCalls[0].Name)
	assert.Equal(t, "container", unitManager.StartCalls[0].UnitType)
	assert.Equal(t, "api", unitManager.StartCalls[1].Name)
}

// TestUpCommand_WithOutput demonstrates proper output capture using helpers.
func TestUpCommand_WithOutput(t *testing.T) {
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
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute with full output capture
	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	// Verify success
	require.NoError(t, err)

	// Verify service calls
	assert.Len(t, unitManager.StartCalls, 1)
	assert.Equal(t, "web", unitManager.StartCalls[0].Name)

	// Verify output captured correctly
	assert.Contains(t, output, "Starting 1 units")
	assert.Contains(t, output, "Successfully started 1 units")
}
