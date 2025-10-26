/*
Copyright © 2025 Travis Lyons travis.lyons@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
)

// PullOptions holds pull command options.
type PullOptions struct {
	// No flags currently supported
}

// PullDeps holds pull dependencies.
type PullDeps struct {
	CommonDeps
	ExecCommand func(ctx context.Context, name string, arg ...string) *exec.Cmd
	Environ     func() []string
	Getuid      func() int
}

// PullCommand represents the pull command.
type PullCommand struct{}

// NewPullCommand creates a new PullCommand.
func NewPullCommand() *PullCommand {
	return &PullCommand{}
}

// getApp retrieves the App from the command context.
func (c *PullCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand gets the cobra command.
func (c *PullCommand) GetCobraCommand() *cobra.Command {
	var opts PullOptions

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "pull an image from a registry",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return pullCmd
}

// buildDeps creates production dependencies for pull.
func (c *PullCommand) buildDeps(app *App) PullDeps {
	return PullDeps{
		CommonDeps:  NewRootDeps(app),
		ExecCommand: exec.CommandContext,
		Environ:     os.Environ,
		Getuid:      os.Getuid,
	}
}

// Run executes the pull command with injected dependencies.
func (c *PullCommand) Run(ctx context.Context, app *App, _ PullOptions, deps PullDeps, args []string) error {
	if len(args) == 0 {
		for _, repoConfig := range app.Config.Repositories {
			gitRepo := git.NewGitRepository(repoConfig, app.ConfigProvider)
			composeDir := gitRepo.Path
			if repoConfig.ComposeDir != "" {
				composeDir = filepath.Join(gitRepo.Path, repoConfig.ComposeDir)
			}

			projects, err := compose.ReadProjects(composeDir)
			if err != nil {
				deps.Logger.Error("Failed to read projects from repository", "name", repoConfig.Name, "composeDir", repoConfig.ComposeDir, "error", err)
				deps.Logger.Info("Check that the composeDir path exists in the repository", "repository", repoConfig.Name, "expectedPath", repoConfig.ComposeDir)
				continue
			}

			for _, project := range projects {
				for _, service := range project.Services {
					if err := c.pullImage(ctx, app, deps, service.Image); err != nil {
						deps.Logger.Error("Failed to pull image", "image", service.Image, "error", err)
					}
				}
			}
		}
	}
	return nil
}

func (c *PullCommand) pullImage(ctx context.Context, app *App, deps PullDeps, image string) error {
	// Use podman pull directly - it handles rootless mode automatically
	args := []string{"pull"}

	// Always show progress for better user experience
	// Only add quiet flag if explicitly not verbose
	if !app.Config.Verbose {
		args = append(args, "--quiet")
	}

	args = append(args, image)

	// Build command safely - podman is a known safe command
	cmd := deps.ExecCommand(ctx, "podman", args...) // #nosec G204

	// Set up environment for rootless operation
	if app.Config.UserMode {
		env := deps.Environ()
		// Ensure XDG_RUNTIME_DIR is set for rootless operation
		if os.Getenv("XDG_RUNTIME_DIR") == "" {
			env = append(env, fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", deps.Getuid()))
		}
		cmd.Env = env
	}

	// Show progress by connecting stdout/stderr to the current process
	if app.Config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		deps.Logger.Info("Pulling image", "image", image)
	} else {
		// For non-verbose mode, still capture output for error reporting
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("podman pull failed: %w\nOutput: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}

	// Run the command and wait for completion
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("podman pull failed: %w", err)
	}

	deps.Logger.Info("Successfully pulled image", "image", image)
	return nil
}
