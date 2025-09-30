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
	assert.Contains(t, output, "quad-ops")
}

// TestVersionCommand_OutputContainsVersionInfo tests version output details.
func TestVersionCommand_OutputContainsVersionInfo(t *testing.T) {
	versionCmd := NewVersionCommand()
	cmd := versionCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version")
	assert.Contains(t, output, "commit:")
	assert.Contains(t, output, "built:")
	assert.Contains(t, output, "go:")
}

// TestVersionCommand_Help tests help output.
func TestVersionCommand_Help(t *testing.T) {
	cmd := NewVersionCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Show version information")
}

// TestVersionCommand_DevVersion tests development version handling.
func TestVersionCommand_DevVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "dev"

	versionCmd := NewVersionCommand()
	cmd := versionCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version dev")
	assert.Contains(t, output, "Skipping update check for development build")
}

// TestVersionCommand_ReleaseVersion tests release version handling.
func TestVersionCommand_ReleaseVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v1.0.0"

	versionCmd := NewVersionCommand()
	cmd := versionCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops version v1.0.0")
	assert.Contains(t, output, "Checking for updates")
}

// TestVersionCommand_BuildInfo tests build information display.
func TestVersionCommand_BuildInfo(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	defer func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	}()

	Version = "v1.2.3"
	Commit = "abc123"
	Date = "2025-01-01"

	versionCmd := NewVersionCommand()
	cmd := versionCmd.GetCobraCommand()

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	assert.Contains(t, output, "v1.2.3")
	assert.Contains(t, output, "abc123")
	assert.Contains(t, output, "2025-01-01")
}
