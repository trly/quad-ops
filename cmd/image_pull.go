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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/log"
)

// PullCommand represents the pull command.
type PullCommand struct{}

// GetCobraCommand gets the cobracomman.
func (c *PullCommand) GetCobraCommand() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "pull an image from a registry",
		Args:  cobra.MaximumNArgs(1),
	}

	pullCmd.Run = func(_ *cobra.Command, args []string) {
		if len(args) == 0 {
			for _, repoConfig := range cfg.Repositories {
				gitRepo := git.NewGitRepository(repoConfig)
				composeDir := gitRepo.Path
				if repoConfig.ComposeDir != "" {
					composeDir = filepath.Join(gitRepo.Path, repoConfig.ComposeDir)
				}

				projects, err := compose.ReadProjects(composeDir)
				if err != nil {
					log.GetLogger().Error("Failed to read projects from repository", "name", repoConfig.Name, "composeDir", repoConfig.ComposeDir, "error", err)
					log.GetLogger().Info("Check that the composeDir path exists in the repository", "repository", repoConfig.Name, "expectedPath", repoConfig.ComposeDir)
				}

				for _, project := range projects {
					for _, service := range project.Services {
						err := pullImage(service.Image)
						if err != nil {
							log.GetLogger().Error("Failed to pull image", "image", service.Image, "error", err)
						}
					}
				}
			}
		}
	}
	return pullCmd
}

func pullImage(image string) error {
	// Use podman pull directly - it handles rootless mode automatically
	args := []string{"pull"}

	// Always show progress for better user experience
	// Only add quiet flag if explicitly not verbose
	if !cfg.Verbose {
		args = append(args, "--quiet")
	}

	args = append(args, image)

	// Build command safely - podman is a known safe command
	cmd := exec.Command("podman", args...) // #nosec G204

	// Set up environment for rootless operation
	if cfg.UserMode {
		env := os.Environ()
		// Ensure XDG_RUNTIME_DIR is set for rootless operation
		if os.Getenv("XDG_RUNTIME_DIR") == "" {
			env = append(env, fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", os.Getuid()))
		}
		cmd.Env = env
	}

	// Show progress by connecting stdout/stderr to the current process
	if cfg.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.GetLogger().Info("Pulling image", "image", image)
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

	log.GetLogger().Info("Successfully pulled image", "image", image)
	return nil
}
