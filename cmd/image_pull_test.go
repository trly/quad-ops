package cmd

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestPullCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return assert.AnError
			},
		}).
		Build(t)

	cmd := NewPullCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
}

func TestPullCommand_Success(t *testing.T) {
	app := NewAppBuilder(t).Build(t)

	cmd := NewPullCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{})
	assert.NoError(t, err)
}

func TestPullCommand_Help(t *testing.T) {
	cmd := NewPullCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "pull an image from a registry")
}

func TestPullCommand_Run_WithMockDeps(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	pullCmd := NewPullCommand()

	// Mock dependencies
	var executedCommands [][]string
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			// Track executed commands
			allArgs := append([]string{name}, args...)
			executedCommands = append(executedCommands, allArgs)

			// Return a command that will succeed
			return exec.Command("echo", "pulled successfully")
		},
		Environ: func() []string {
			return []string{"TEST=true"}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.Run(ctx, app, PullOptions{}, deps, []string{})

	assert.NoError(t, err)
	// Since there are no repositories configured, no commands should be executed
	assert.Empty(t, executedCommands)
}

func TestPullCommand_Flags(t *testing.T) {
	cmd := NewPullCommand().GetCobraCommand()

	// Verify command has no specific flags (uses parent image command flags)
	flags := cmd.Flags()
	assert.Equal(t, 0, flags.NFlag(), "Pull command should not have specific flags")
}
