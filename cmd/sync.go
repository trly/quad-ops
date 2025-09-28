// Package cmd provides the command line interface for quad-ops
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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
)

// SyncCommand represents the sync command for quad-ops CLI.
type SyncCommand struct{}

// NewSyncCommand creates a new SyncCommand.
func NewSyncCommand() *SyncCommand {
	return &SyncCommand{}
}

// getApp retrieves the App from the command context.
func (c *SyncCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

var (
	dryRun   bool
	repoName string
	force    bool
)

// GetCobraCommand returns the cobra command for sync operations.
func (c *SyncCommand) GetCobraCommand() *cobra.Command {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes the Docker Compose files defined in configured repositories with quadlet units on the local system.",
		Long: `Synchronizes the Docker Compose files defined in configured repositories with quadlet units on the local system.

Repositories are defined in the quad-ops config file as a list of Repository objects.

---
repositories:
  - name: quad-ops-compose
    url: https://github.com/trly/quad-ops-compose.git
    target: main
    cleanup:
      action: Delete`,

		PreRun: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Validate system requirements for sync operations
			if err := app.Validator.SystemRequirements(); err != nil {
				app.Logger.Error("System requirements not met", "error", err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			if err := os.MkdirAll(app.Config.QuadletDir, 0750); err != nil {
				app.Logger.Error("Failed to create quadlet directory", "error", err)
				os.Exit(1)
			}

			c.syncRepositories(app)
		},
	}

	syncCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Perform a dry run without making any changes.")
	syncCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Synchronize a single, named, repository.")
	syncCmd.Flags().BoolVarP(&force, "force", "f", false, "Force synchronization even if the repository has not changed.")

	return syncCmd
}
func (c *SyncCommand) syncRepositories(app *App) {
	// Create a shared map to track processed units across all repositories
	processedUnits := make(map[string]bool)
	for _, repoConfig := range app.Config.Repositories {
		if repoName != "" && repoConfig.Name != repoName {
			app.Logger.Debug("Skipping repository as it does not match the specified name", "repo", repoConfig.Name)
			continue
		}

		if !dryRun {
			app.Logger.Debug("Processing repository", "name", repoConfig.Name)

			gitRepo := git.NewGitRepository(repoConfig, app.ConfigProvider)
			if err := gitRepo.SyncRepository(); err != nil {
				app.Logger.Error("Failed to sync repository", "name", repoConfig.Name, "error", err)
				continue
			}

			// Determine compose directory path
			composeDir := gitRepo.Path
			if repoConfig.ComposeDir != "" {
				composeDir = filepath.Join(gitRepo.Path, repoConfig.ComposeDir)
			}

			app.Logger.Debug("Looking for compose files", "dir", composeDir)

			projects, err := compose.ReadProjects(composeDir)
			if err != nil {
				if repoConfig.ComposeDir != "" {
					app.Logger.Error("Failed to read projects from repository", "name", repoConfig.Name, "composeDir", repoConfig.ComposeDir, "error", err)
					app.Logger.Info("Check that the composeDir path exists in the repository", "repository", repoConfig.Name, "expectedPath", repoConfig.ComposeDir)
				} else {
					app.Logger.Error("Failed to read projects from repository", "name", repoConfig.Name, "error", err)
				}
				continue
			}

			// Process projects with the shared map, only perform cleanup after the last repository
			isLastRepo := repoConfig.Name == app.Config.Repositories[len(app.Config.Repositories)-1].Name

			// If specific repo is specified, always do cleanup
			if repoName != "" {
				isLastRepo = true
			}

			processor := compose.NewDefaultProcessor(force)
			if processedUnits != nil {
				processor.WithExistingProcessedUnits(processedUnits)
			}

			err = processor.ProcessProjects(projects, isLastRepo)
			if err != nil {
				app.Logger.Error("Failed to process projects from repository", "name", repoConfig.Name, "error", err)
				continue
			}

			updatedMap := processor.GetProcessedUnits()

			// Update the shared map with units from this repository
			processedUnits = updatedMap
		} else {
			app.Logger.Info("Dry-run: would process repository", "name", repoConfig.Name)
		}
	}
}
