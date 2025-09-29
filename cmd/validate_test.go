package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCommand_Basic tests validate command.
func TestValidateCommand_Basic(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	// Should contain validation results - output might be empty if no files found
	if output != "" {
		assert.Contains(t, output, "Validating")
	}
}
