package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
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

	var executedCommands [][]string
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			allArgs := append([]string{name}, args...)
			executedCommands = append(executedCommands, allArgs)
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
	assert.Empty(t, executedCommands)
}

func TestPullCommand_Flags(t *testing.T) {
	cmd := NewPullCommand().GetCobraCommand()

	flags := cmd.Flags()
	assert.Equal(t, 0, flags.NFlag(), "Pull command should not have specific flags")
}

// TestPullCommand_Run_WithRepositories is skipped - requires actual git repository setup.
func TestPullCommand_Run_WithRepositories(t *testing.T) {
	t.Skip("Requires complex repository setup with compose files - covered by integration tests")
}

func TestPullCommand_PullImage_Verbose(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose: true,
		}).
		Build(t)

	pullCmd := NewPullCommand()

	var commandCalled bool
	var argsReceived []string
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			commandCalled = true
			argsReceived = args
			assert.Equal(t, "podman", name)
			cmd := exec.CommandContext(context.Background(), "echo", "pulled")
			return cmd
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.pullImage(ctx, app, deps, "nginx:latest")

	assert.NoError(t, err)
	assert.True(t, commandCalled)
	assert.Contains(t, argsReceived, "pull")
	assert.Contains(t, argsReceived, "nginx:latest")
	assert.NotContains(t, argsReceived, "--quiet")
}

func TestPullCommand_PullImage_NonVerbose(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose: false,
		}).
		Build(t)

	pullCmd := NewPullCommand()

	var commandCalled bool
	var argsReceived []string
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			commandCalled = true
			argsReceived = args
			assert.Equal(t, "podman", name)
			cmd := exec.CommandContext(context.Background(), "echo", "pulled")
			return cmd
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.pullImage(ctx, app, deps, "nginx:latest")

	assert.NoError(t, err)
	assert.True(t, commandCalled)
	assert.Contains(t, argsReceived, "pull")
	assert.Contains(t, argsReceived, "--quiet")
	assert.Contains(t, argsReceived, "nginx:latest")
}

func TestPullCommand_PullImage_UserMode(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			UserMode: true,
			Verbose:  false,
		}).
		Build(t)

	pullCmd := NewPullCommand()

	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(context.Background(), "echo", "pulled")
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	originalXDG := os.Getenv("XDG_RUNTIME_DIR")
	_ = os.Unsetenv("XDG_RUNTIME_DIR")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_RUNTIME_DIR", originalXDG)
		}
	}()

	ctx := context.Background()
	err := pullCmd.pullImage(ctx, app, deps, "nginx:latest")

	assert.NoError(t, err)
}

func TestPullCommand_PullImage_Failed(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose: false,
		}).
		Build(t)

	pullCmd := NewPullCommand()

	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.pullImage(ctx, app, deps, "nonexistent:image")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "podman pull failed")
}

func TestPullCommand_Run_RepositoryReadError(t *testing.T) {
	tempDir := t.TempDir()

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose:       false,
			RepositoryDir: tempDir,
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://github.com/test/repo.git", ComposeDir: "nonexistent"},
			},
		}).
		Build(t)

	pullCmd := NewPullCommand()
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			return exec.Command("echo", "pulled")
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.Run(ctx, app, PullOptions{}, deps, []string{})

	assert.NoError(t, err)
}

func TestPullCommand_Run_PullImageError(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(repoDir, 0750)
	require.NoError(t, err)

	composeContent := `
version: "3.8"
services:
  web:
    image: nginx:latest
`
	err = os.WriteFile(filepath.Join(repoDir, "docker-compose.yml"), []byte(composeContent), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose:       false,
			RepositoryDir: tempDir,
			Repositories: []config.Repository{
				{Name: "test-repo", URL: "https://github.com/test/repo.git"},
			},
		}).
		Build(t)

	pullCmd := NewPullCommand()

	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err = pullCmd.Run(ctx, app, PullOptions{}, deps, []string{})

	assert.NoError(t, err)
}

func TestPullCommand_BuildDeps(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	pullCmd := NewPullCommand()

	deps := pullCmd.buildDeps(app)

	assert.NotNil(t, deps.ExecCommand)
	assert.NotNil(t, deps.Environ)
	assert.NotNil(t, deps.Getuid)
	assert.NotNil(t, deps.Logger)
	assert.NotNil(t, deps.FileSystem.Stat)
}

func TestPullCommand_PullImage_VerboseError(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose: true,
		}).
		Build(t)

	pullCmd := NewPullCommand()

	var runCalled bool
	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			runCalled = true
			cmd := exec.Command("sh", "-c", "exit 1")
			return cmd
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.pullImage(ctx, app, deps, "failing:image")

	assert.Error(t, err)
	assert.True(t, runCalled)
	assert.Contains(t, err.Error(), "podman pull failed")
}

func TestPullCommand_GetApp(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	pullCmd := NewPullCommand()
	cmd := pullCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	retrievedApp := pullCmd.getApp(cmd)
	assert.Equal(t, app, retrievedApp)
}

func TestPullCommand_Run_WithArgs(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	pullCmd := NewPullCommand()

	deps := PullDeps{
		CommonDeps: NewCommonDeps(testutil.NewTestLogger(t)),
		ExecCommand: func(_ string, _ ...string) *exec.Cmd {
			return exec.Command("echo", "pulled")
		},
		Environ: func() []string {
			return []string{}
		},
		Getuid: func() int {
			return 1000
		},
	}

	ctx := context.Background()
	err := pullCmd.Run(ctx, app, PullOptions{}, deps, []string{"nginx:latest"})

	assert.NoError(t, err)
}

// TestPullCommand_PullImage_CancelSupport is skipped - testing Cancel requires CommandContext.
func TestPullCommand_PullImage_CancelSupport(t *testing.T) {
	t.Skip("Cancel support requires complex CommandContext setup - covered by integration tests")
}
