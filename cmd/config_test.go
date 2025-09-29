package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
)

// TestConfigCommand_DisplayConfig tests config command.
func TestConfigCommand_DisplayConfig(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			QuadletDir:    "/test/quadlet",
			RepositoryDir: "/test/repos",
			UserMode:      false,
			Verbose:       true,
		}).
		Build(t)

	configCmd := NewConfigCommand()
	cmd := configCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	assert.Contains(t, output, "/test/quadlet")
	assert.Contains(t, output, "/test/repos")
}
