package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommandFlags verifies flag parsing.
func TestRootCommandFlags(t *testing.T) {
	rootCmd := &RootCommand{}
	cmd := rootCmd.GetCobraCommand()

	// Test flag defaults
	userFlag := cmd.PersistentFlags().Lookup("user")
	require.NotNil(t, userFlag)
	assert.Equal(t, "false", userFlag.DefValue)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)

	configFlag := cmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, "", configFlag.DefValue)

	quadletDirFlag := cmd.PersistentFlags().Lookup("quadlet-dir")
	require.NotNil(t, quadletDirFlag)
	assert.Equal(t, "", quadletDirFlag.DefValue)

	repositoryDirFlag := cmd.PersistentFlags().Lookup("repository-dir")
	require.NotNil(t, repositoryDirFlag)
	assert.Equal(t, "", repositoryDirFlag.DefValue)

	outputFlag := cmd.PersistentFlags().Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, "text", outputFlag.DefValue)
}

func TestRootCommand_PersistentPreRun_InvalidRepositoryDir(t *testing.T) {
	rootCmd := &RootCommand{}
	deps := RootDeps{
		ValidatePath: func(path string) error {
			if path == "/invalid/path" {
				return errors.New("path validation failed")
			}
			return nil
		},
		ExpandEnv: func(s string) string { return s },
	}

	opts := RootOptions{
		RepositoryDir: "/invalid/path",
	}

	cmd := rootCmd.GetCobraCommand()
	cmd.SetContext(context.Background())
	err := rootCmd.persistentPreRun(cmd, opts, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repository directory")
}

func TestRootCommand_PersistentPreRun_InvalidQuadletDir(t *testing.T) {
	rootCmd := &RootCommand{}
	deps := RootDeps{
		ValidatePath: func(path string) error {
			if path == "/invalid/quadlet" {
				return errors.New("path validation failed")
			}
			return nil
		},
		ExpandEnv: func(s string) string { return s },
	}

	opts := RootOptions{
		QuadletDir: "/invalid/quadlet",
	}

	cmd := rootCmd.GetCobraCommand()
	cmd.SetContext(context.Background())
	err := rootCmd.persistentPreRun(cmd, opts, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quadlet directory")
}

func TestRootCommand_PersistentPreRun_Success(t *testing.T) {
	rootCmd := &RootCommand{}
	deps := RootDeps{
		ValidatePath: func(_ string) error { return nil },
		ExpandEnv:    func(s string) string { return s },
	}

	opts := RootOptions{
		Verbose:       true,
		UserMode:      true,
		RepositoryDir: "/valid/repo",
		QuadletDir:    "/valid/quadlet",
	}

	cmd := rootCmd.GetCobraCommand()
	cmd.SetContext(context.Background())
	err := rootCmd.persistentPreRun(cmd, opts, deps)
	assert.NoError(t, err)
}

func TestRootCommand_Help(t *testing.T) {
	cmd := (&RootCommand{}).GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Quad-Ops manages Quadlet container units")
	assert.Contains(t, output, "Available Commands:")
}

func TestRootCommand_SubcommandPresent(t *testing.T) {
	cmd := (&RootCommand{}).GetCobraCommand()

	// Verify key subcommands are present
	syncCmd := findCommand(cmd, "sync")
	assert.NotNil(t, syncCmd, "sync command should be present")

	daemonCmd := findCommand(cmd, "daemon")
	assert.NotNil(t, daemonCmd, "daemon command should be present")

	unitCmd := findCommand(cmd, "unit")
	assert.NotNil(t, unitCmd, "unit command should be present")
}

// findCommand finds a command by name in the command tree.
func findCommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
