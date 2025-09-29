package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSyncCommand_ValidationFailure tests basic sync command functionality.
func TestSyncCommand_ValidationFailure(t *testing.T) {
	// Setup exit capture
	var exitCode int
	oldSyncExit := syncExitFunc
	syncExitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { syncExitFunc = oldSyncExit })

	// Create app with failing validator (simplest failure case)
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	// Create command and setup context
	syncCmd := NewSyncCommand()
	cmd := syncCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute PreRun (which should trigger validation and exit)
	cmd.PreRun(cmd, []string{})

	// Verify exit was called with code 1
	assert.Equal(t, 1, exitCode)
}
