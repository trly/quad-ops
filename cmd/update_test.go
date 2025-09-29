package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateCommand_Help tests update command help.
func TestUpdateCommand_Help(t *testing.T) {
	updateCmd := NewUpdateCommand()
	cmd := updateCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Update quad-ops to the latest version")
}
