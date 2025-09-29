package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDownCommand_ValidationFailure tests down command validation.
func TestDownCommand_ValidationFailure(t *testing.T) {
	// Setup exit capture (down command uses same exitFunc as up)
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

	// Create command and setup context
	downCmd := NewDownCommand()
	cmd := downCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Execute PreRun (which should trigger validation)
	cmd.PreRun(cmd, []string{})

	// Verify exit was called with code 1
	assert.Equal(t, 1, exitCode)
}
