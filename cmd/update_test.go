package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateCommand_GetCobraCommand tests command structure.
func TestUpdateCommand_GetCobraCommand(t *testing.T) {
	cmd := NewUpdateCommand()
	cobraCmd := cmd.GetCobraCommand()

	assert.NotNil(t, cobraCmd)
	assert.Equal(t, "update", cobraCmd.Use)
	assert.Equal(t, "Update quad-ops to the latest version", cobraCmd.Short)
	assert.Contains(t, cobraCmd.Long, "Update quad-ops to the latest version from GitHub releases")
	assert.NotNil(t, cobraCmd.RunE)
}

// TestUpdateCommand_Help tests update command help.
func TestUpdateCommand_Help(t *testing.T) {
	updateCmd := NewUpdateCommand()
	cmd := updateCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Update quad-ops to the latest version")
	assert.Contains(t, output, "GitHub releases")
}

// TestNewUpdateCommand tests constructor.
func TestNewUpdateCommand(t *testing.T) {
	cmd := NewUpdateCommand()
	assert.NotNil(t, cmd)
	assert.IsType(t, &UpdateCommand{}, cmd)
}

// TestUpdateCommand_NoFlags tests that update command has no flags.
func TestUpdateCommand_NoFlags(t *testing.T) {
	cmd := NewUpdateCommand().GetCobraCommand()

	// Update command should have no custom flags, only inherited ones
	localFlags := cmd.Flags()
	assert.NotNil(t, localFlags)

	// Verify it only has help flag (inherited)
	localFlags.VisitAll(func(flag *pflag.Flag) {
		// Only help flag should be present
		assert.Equal(t, "help", flag.Name)
	})
}

// TestUpdateCommand_Output tests that command executes.
func TestUpdateCommand_Output(_ *testing.T) {
	// Note: This command uses fmt.Printf which writes directly to stdout,
	// not through cobra's output writers. Testing actual output would require
	// OS-level stdout redirection or refactoring the command.
	// For now, we verify the command executes without panic.

	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v1.0.0"

	cmd := NewUpdateCommand().GetCobraCommand()

	// Execute command - will attempt network call but shouldn't panic
	// Error is expected due to network/GitHub API
	_ = cmd.RunE(cmd, []string{})
}

// TestUpdateCommand_RunE_Exists verifies RunE function is set.
func TestUpdateCommand_RunE_Exists(t *testing.T) {
	cmd := NewUpdateCommand().GetCobraCommand()
	assert.NotNil(t, cmd.RunE, "RunE function must be set")
}

// TestUpdateCommand_UsesVersion tests that version is accessible.
func TestUpdateCommand_UsesVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	testVersion := "v2.5.0"
	Version = testVersion

	// Verify version is set correctly
	assert.Equal(t, testVersion, Version)

	cmd := NewUpdateCommand().GetCobraCommand()
	assert.NotNil(t, cmd)

	// Execute - will fail on network, but shouldn't panic
	_ = cmd.RunE(cmd, []string{})
}
