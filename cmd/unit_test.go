package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitCommand_Help tests unit command help.
func TestUnitCommand_Help(t *testing.T) {
	unitCmd := NewUnitCommand()
	cmd := unitCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "quadlet units") // "subcommands for managing and viewing quadlet units"
	assert.Contains(t, output, "list")
	assert.Contains(t, output, "show")
	assert.Contains(t, output, "status")
}
