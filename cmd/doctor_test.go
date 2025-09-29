package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDoctorCommand_ValidationFailure tests basic doctor command functionality.
func TestDoctorCommand_ValidationFailure(t *testing.T) {
	// Setup exit capture
	var exitCode int
	oldDoctorExit := doctorExitFunc
	doctorExitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { doctorExitFunc = oldDoctorExit })

	// Create app with failing validator
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	// Create and execute command
	doctorCmd := NewDoctorCommand()
	cmd := doctorCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	_, err := ExecuteCommandWithCapture(t, cmd, []string{})

	// Should exit with error code
	require.NoError(t, err) // Command itself doesn't return error, it exits
	assert.Equal(t, 1, exitCode)
	// Note: Doctor command output capture doesn't work reliably in test environment
}
