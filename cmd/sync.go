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
	"time"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/logger"
	"github.com/trly/quad-ops/internal/unit"
)

// SyncCommand represents the sync command for quad-ops CLI.
type SyncCommand struct{}

var (
	dryRun       bool
	repoName     string
	daemon       bool
	syncInterval time.Duration
	force        bool
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

		Run: func(_ *cobra.Command, _ []string) {
			if err := os.MkdirAll(config.GetConfig().QuadletDir, 0750); err != nil {
				logger.GetLogger().Error("Failed to create quadlet directory", "error", err)
				os.Exit(1)
			}

			if syncInterval > 0 {
				cfg.SyncInterval = syncInterval
			}

			syncRepositories(cfg)

			if daemon {
				syncDaemon(cfg)
			}
		},
	}

	syncCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Perform a dry run without making any changes.")
	syncCmd.Flags().BoolVar(&daemon, "daemon", false, "Run as a daemon.")
	syncCmd.Flags().DurationVarP(&syncInterval, "sync-interval", "i", 5*time.Minute, "Interval between synchronization checks.")
	syncCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Synchronize a single, named, repository.")
	syncCmd.Flags().BoolVarP(&force, "force", "f", false, "Force synchronization even if the repository has not changed.")

	return syncCmd
}
func syncRepositories(cfg *config.Config) {
	// Create a shared map to track processed units across all repositories
	processedUnits := make(map[string]bool)
	for _, repoConfig := range cfg.Repositories {
		if repoName != "" && repoConfig.Name != repoName {
			logger.GetLogger().Debug("Skipping repository as it does not match the specified name", "repo", repoConfig.Name)
			continue
		}

		if !dryRun {
			logger.GetLogger().Info("Processing repository", "name", repoConfig.Name)

			gitRepo := git.NewGitRepository(repoConfig)
			if err := gitRepo.SyncRepository(); err != nil {
				logger.GetLogger().Error("Failed to sync repository", "name", repoConfig.Name, "error", err)
				continue
			}

			// Determine compose directory path
			composeDir := gitRepo.Path
			if repoConfig.ComposeDir != "" {
				composeDir = filepath.Join(gitRepo.Path, repoConfig.ComposeDir)
			}

			logger.GetLogger().Debug("Looking for compose files", "dir", composeDir)

			projects, err := compose.ReadProjects(composeDir)
			if err != nil {
				logger.GetLogger().Error("Failed to read projects from repository", "name", repoConfig.Name, "error", err)
				continue
			}

			// Process projects with the shared map, only perform cleanup after the last repository
			isLastRepo := repoConfig.Name == cfg.Repositories[len(cfg.Repositories)-1].Name

			// If specific repo is specified, always do cleanup
			if repoName != "" {
				isLastRepo = true
			}

			updatedMap, err := unit.ProcessComposeProjects(projects, force, processedUnits, isLastRepo)
			if err != nil {
				logger.GetLogger().Error("Failed to process projects from repository", "name", repoConfig.Name, "error", err)
				continue
			}

			// Update the shared map with units from this repository
			processedUnits = updatedMap
		} else {
			logger.GetLogger().Info("Dry-run: would process repository", "name", repoConfig.Name)
		}
	}
}

func syncDaemon(cfg *config.Config) {
	logger.GetLogger().Info("Starting sync daemon", "interval", cfg.SyncInterval)
	ticker := time.NewTicker(cfg.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		logger.GetLogger().Info("Starting scheduled sync")
		syncRepositories(cfg)
	}
}
