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
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/unit"
)

type SyncCommand struct{}

var (
	dryRun       bool
	repoName     string
	daemon       bool
	syncInterval time.Duration
	force        bool
)

func (c *SyncCommand) GetCobraCommand() *cobra.Command {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes the manifests defined in configured repositories with quadlet units on the local system.",
		Long: `Synchronizes the manifests defined in configured repositories with quadlet units on the local system.

Repositories are defined in the quad-ops config file as a list of Repository objects.

---
repositories:
  - name: quad-ops-manifests
    url: https://github.com/trly/quad-ops-manifests.git
    target: main
    cleanup:
      action: Delete`,

		Run: func(cmd *cobra.Command, args []string) {
			if err := os.MkdirAll(config.GetConfig().QuadletDir, 0755); err != nil {
				log.Fatal("Failed to create quadlet directory:", err)
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
	for _, repoConfig := range cfg.Repositories {
		if repoName != "" && repoConfig.Name != repoName {
			if config.GetConfig().Verbose {
				log.Printf("skipping repository %s as it does not match the specified repository name", repoConfig.Name)
			}
			continue
		}

		if !dryRun {
			if config.GetConfig().Verbose {
				log.Printf("processing repository: %s", repoConfig.Name)
			}

			repo := git.NewRepository(repoConfig)
			if err := repo.SyncRepository(); err != nil {
				log.Printf("error syncing repository %s: %v", repoConfig.Name, err)
				continue
			}

			if err := unit.ProcessManifests(repo, force); err != nil {
				log.Printf("error processing manifests for %s: %v", repoConfig.Name, err)
				continue
			}
		} else {
			log.Printf("dry-run: would process repository: %s", repoConfig.Name)
		}
	}
}

func syncDaemon(cfg *config.Config) {
	log.Printf("starting sync daemon with interval: %v", cfg.SyncInterval)
	ticker := time.NewTicker(cfg.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Printf("starting scheduled sync")
		syncRepositories(cfg)
	}
}
