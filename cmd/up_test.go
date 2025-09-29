package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/repository"
)

// TestUpCommand_ValidationFailure verifies that validation failures are handled correctly.
func TestUpCommand_ValidationFailure(t *testing.T) {
	// Setup exit capture
	var exitCode int
	oldExit := exitFunc
	exitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { exitFunc = oldExit })

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
	ctx := context.WithValue(context.Background(), appContextKey, app)
	cmd.SetContext(ctx)

	// Execute PreRun (which should trigger validation)
	cmd.PreRun(cmd, []string{})

	// Verify exit was called with code 1
	assert.Equal(t, 1, exitCode)
}

// TestUpCommand_StartUnitsSuccess verifies successful unit starting.
func TestUpCommand_StartUnitsSuccess(t *testing.T) {
	// Setup exit capture
	var exitCode int
	oldExit := exitFunc
	exitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { exitFunc = oldExit })

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

	// Setup command with captured output
	upCmd := NewUpCommand()
	cmd := upCmd.GetCobraCommand()
	ctx := context.WithValue(context.Background(), appContextKey, app)
	cmd.SetContext(ctx)

	// Execute command (both PreRun and Run)
	cmd.PreRun(cmd, []string{})
	cmd.Run(cmd, []string{})

	// Verify no exit was called (success case)
	assert.Equal(t, 0, exitCode)

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
	// Setup exit capture
	var exitCode int
	oldExit := exitFunc
	exitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { exitFunc = oldExit })

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

	// Verify success (no exit called)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	// Verify service calls
	assert.Len(t, unitManager.StartCalls, 1)
	assert.Equal(t, "web", unitManager.StartCalls[0].Name)

	// Verify output captured correctly
	assert.Contains(t, output, "Starting 1 units")
	assert.Contains(t, output, "Successfully started 1 units")
}
