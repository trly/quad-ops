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
	"context"
	"fmt"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
)

// SyncOptions holds sync command options.
type SyncOptions struct {
	DryRun   bool
	RepoName string
	Force    bool
}

// SyncDeps holds sync dependencies.
type SyncDeps struct {
	CommonDeps
	NewGitRepository    func(repository config.Repository, configProvider config.Provider) *git.Repository
	ReadProjects        func(baseDir string) ([]*types.Project, error)
	NewDefaultProcessor func(force bool) *compose.Processor
}

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

// GetCobraCommand returns the cobra command for sync operations.
func (c *SyncCommand) GetCobraCommand() *cobra.Command {
	var opts SyncOptions

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

		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	syncCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "Perform a dry run without making any changes.")
	syncCmd.Flags().StringVarP(&opts.RepoName, "repo", "r", "", "Synchronize a single, named, repository.")
	syncCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force synchronization even if the repository has not changed.")

	return syncCmd
}

// buildDeps creates production dependencies for the sync command.
func (c *SyncCommand) buildDeps(app *App) SyncDeps {
	return SyncDeps{
		CommonDeps:          NewRootDeps(app),
		NewGitRepository:    git.NewGitRepository,
		ReadProjects:        compose.ReadProjects,
		NewDefaultProcessor: compose.NewDefaultProcessor,
	}
}

// Run executes the sync command with injected dependencies.
func (c *SyncCommand) Run(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps) error {
	// Ensure quadlet directory exists
	if err := deps.FileSystem.MkdirAll(app.Config.QuadletDir, 0750); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	return c.syncRepositories(ctx, app, opts, deps)
}

// syncRepositories performs the actual repository synchronization.
func (c *SyncCommand) syncRepositories(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps) error {
	// Create a shared map to track processed units across all repositories
	processedUnits := make(map[string]bool)

	for _, repoConfig := range app.Config.Repositories {
		if opts.RepoName != "" && repoConfig.Name != opts.RepoName {
			deps.Logger.Debug("Skipping repository as it does not match the specified name", "repo", repoConfig.Name)
			continue
		}

		if err := c.processRepository(ctx, app, repoConfig, opts, deps, processedUnits); err != nil {
			deps.Logger.Error("Failed to process repository", "name", repoConfig.Name, "error", err)
			// Continue processing other repositories
		}
	}

	return nil
}

// processRepository processes a single repository.
func (c *SyncCommand) processRepository(_ context.Context, app *App, repoConfig config.Repository, opts SyncOptions, deps SyncDeps, _ map[string]bool) error {
	if !opts.DryRun {
		deps.Logger.Debug("Processing repository", "name", repoConfig.Name)

		gitRepo := deps.NewGitRepository(repoConfig, app.ConfigProvider)
		if err := gitRepo.SyncRepository(); err != nil {
			return fmt.Errorf("failed to sync repository: %w", err)
		}

		// Check if repository content has changed
		// Note: HasChanges method needs implementation in git.Repository
		if !opts.Force {
			// Skip change detection for now - always process when not forced
			deps.Logger.Debug("Change detection not yet implemented, processing repository", "name", repoConfig.Name)
		}

		// Create the target directory if it doesn't exist
		// TODO: Fix config to include CacheDir field
		targetDir := filepath.Join(app.Config.RepositoryDir, repoConfig.Name)
		if err := deps.FileSystem.MkdirAll(targetDir, 0750); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		// Read projects from the repository
		projects, err := deps.ReadProjects(targetDir)
		if err != nil {
			return fmt.Errorf("failed to read projects: %w", err)
		}

		// Process projects using the old processor interface for now
		processor := deps.NewDefaultProcessor(opts.Force)
		// TODO: Update processor interface to use dependency injection
		_ = processor // For now, just acknowledge we have it
		deps.Logger.Info("Would process projects", "count", len(projects))
	}

	return nil
}
