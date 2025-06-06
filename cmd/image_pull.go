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
	"path/filepath"

	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/podman"
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
	conn := podman.GetConnection()
	_, err := images.Pull(conn, image, nil)
	if err != nil {
		return err
	}
	return nil
}
