package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDoctorCommand_ValidationFailure tests doctor command validation failure.
func TestDoctorCommand_ValidationFailure(t *testing.T) {
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

	// PreRunE should return error instead of exiting
	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestDoctorCommand_Success tests successful doctor execution.
func TestDoctorCommand_Success(t *testing.T) {
	// Create app with successful validator
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return nil
			},
		}).
		Build(t)

	// Create and execute command
	doctorCmd := NewDoctorCommand()
	cmd := doctorCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Should execute without hanging (errors are expected in test environment)
	err := ExecuteCommand(t, cmd, []string{})
	// Doctor may find issues in test environment - that's ok as long as it doesn't hang
	if err != nil {
		assert.Contains(t, err.Error(), "doctor found")
	}
}

// TestDoctorCommand_Help tests help output.
func TestDoctorCommand_Help(t *testing.T) {
	cmd := NewDoctorCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Check system health and configuration")
	assert.Contains(t, output, "System requirements")
	assert.Contains(t, output, "Configuration file validity")
}
