package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImageCommand_Help tests image command help.
func TestImageCommand_Help(t *testing.T) {
	imageCmd := NewImageCommand()
	cmd := imageCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "images") // "subcommands for managing and viewing images"
	assert.Contains(t, output, "pull")
}
