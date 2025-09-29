package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionCommand_Basic tests version command.
func TestVersionCommand_Basic(t *testing.T) {
	versionCmd := NewVersionCommand()
	cmd := versionCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	// Should contain version information
	assert.Contains(t, output, "quad-ops")
}
