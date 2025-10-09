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

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
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
	GitSyncer        GitSyncerInterface
	ComposeProcessor ComposeProcessorInterface
	Renderer         RendererInterface
	ArtifactStore    ArtifactStoreInterface
	Lifecycle        LifecycleInterface
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
// Note: Platform-specific dependencies (Renderer, Lifecycle) are obtained via lazy getters in Run().
func (c *SyncCommand) buildDeps(app *App) SyncDeps {
	return SyncDeps{
		CommonDeps:       NewRootDeps(app),
		GitSyncer:        app.GitSyncer,
		ComposeProcessor: app.ComposeProcessor,
		Renderer:         nil, // Obtained via app.GetRenderer(ctx) in Run()
		ArtifactStore:    app.ArtifactStore,
		Lifecycle:        nil, // Obtained via app.GetLifecycle(ctx) in Run()
	}
}

// Run executes the sync command with injected dependencies.
// This method orchestrates components directly following the new architecture pattern:
// GitSyncer → ComposeProcessor → Renderer → ArtifactStore → Lifecycle.
func (c *SyncCommand) Run(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps) error {
	// Get platform-specific components via lazy getters
	renderer, err := app.GetRenderer(ctx)
	if err != nil {
		return fmt.Errorf("platform not supported: %w", err)
	}
	deps.Renderer = renderer

	lifecycle, err := app.GetLifecycle(ctx)
	if err != nil {
		return fmt.Errorf("platform not supported: %w", err)
	}
	deps.Lifecycle = lifecycle

	// Ensure quadlet directory exists
	if err := deps.FileSystem.MkdirAll(app.Config.QuadletDir, 0750); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	// Handle dry-run mode early
	if opts.DryRun {
		deps.Logger.Info("Dry-run mode enabled - no changes will be made")
		return nil
	}

	// 1. Sync git repositories
	deps.Logger.Debug("Syncing git repositories", "count", len(app.Config.Repositories))

	// Filter repositories if specific repo requested
	reposToSync := app.Config.Repositories
	if opts.RepoName != "" {
		reposToSync = make([]config.Repository, 0, 1)
		for _, repo := range app.Config.Repositories {
			if repo.Name == opts.RepoName {
				reposToSync = append(reposToSync, repo)
				break
			}
		}
		if len(reposToSync) == 0 {
			return fmt.Errorf("repository not found: %s", opts.RepoName)
		}
	}

	results, err := deps.GitSyncer.SyncAll(ctx, reposToSync)
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	// Track all services to restart
	servicesToRestart := make(map[string]bool)
	anyChanges := false

	// 2. Process each repository
	for _, result := range results {
		// Check for errors
		if result.Error != nil {
			deps.Logger.Error("Repository sync failed", "repo", result.Repository.Name, "error", result.Error)
			continue
		}

		// Skip if no changes and not forced
		if !result.Changed && !opts.Force {
			deps.Logger.Debug("Repository unchanged, skipping", "repo", result.Repository.Name)
			continue
		}

		deps.Logger.Info("Processing repository", "repo", result.Repository.Name, "changed", result.Changed)

		// 3. Process compose files to service specs
		repoPath := filepath.Join(app.Config.RepositoryDir, result.Repository.Name)

		// Read compose files from repository
		projects, err := compose.ReadProjects(repoPath)
		if err != nil {
			deps.Logger.Error("Failed to read compose projects", "repo", result.Repository.Name, "error", err)
			continue
		}

		if len(projects) == 0 {
			deps.Logger.Debug("No compose projects found", "repo", result.Repository.Name)
			continue
		}

		// Process all compose projects to service specs
		for _, project := range projects {
			specs, err := deps.ComposeProcessor.Process(ctx, project)
			if err != nil {
				deps.Logger.Error("Failed to process compose project",
					"repo", result.Repository.Name, "project", project.Name, "error", err)
				continue
			}

			deps.Logger.Debug("Processed compose project",
				"repo", result.Repository.Name, "project", project.Name, "services", len(specs))

			// 4. Render to platform-specific artifacts
			renderResult, err := deps.Renderer.Render(ctx, specs)
			if err != nil {
				deps.Logger.Error("Failed to render artifacts",
					"repo", result.Repository.Name, "project", project.Name, "error", err)
				continue
			}

			// 5. Write artifacts to disk (with change detection)
			changedPaths, err := deps.ArtifactStore.Write(ctx, renderResult.Artifacts)
			if err != nil {
				deps.Logger.Error("Failed to write artifacts",
					"repo", result.Repository.Name, "project", project.Name, "error", err)
				continue
			}

			if len(changedPaths) > 0 {
				anyChanges = true
				deps.Logger.Info("Artifacts written",
					"repo", result.Repository.Name, "project", project.Name, "changed", len(changedPaths))

				// Track services that need restart
				c.trackChangedServices(changedPaths, renderResult.ServiceChanges, opts.Force, servicesToRestart)
			} else {
				deps.Logger.Debug("No artifact changes", "repo", result.Repository.Name, "project", project.Name)
			}
		}
	}

	// 6. Reload service manager if any artifacts changed
	if anyChanges || opts.Force {
		deps.Logger.Info("Reloading service manager")
		if err := deps.Lifecycle.Reload(ctx); err != nil {
			return fmt.Errorf("failed to reload service manager: %w", err)
		}

		// 7. Restart changed services
		if len(servicesToRestart) > 0 {
			serviceNames := make([]string, 0, len(servicesToRestart))
			for name := range servicesToRestart {
				serviceNames = append(serviceNames, name)
			}

			deps.Logger.Info("Restarting changed services", "count", len(serviceNames))

			// Use RestartMany for dependency-aware restart
			restartErrors := deps.Lifecycle.RestartMany(ctx, serviceNames)

			// Log any restart failures
			for serviceName, err := range restartErrors {
				if err != nil {
					deps.Logger.Error("Failed to restart service", "service", serviceName, "error", err)
				} else {
					deps.Logger.Info("Service restarted", "service", serviceName)
				}
			}
		}
	} else {
		deps.Logger.Info("No changes detected")
	}

	return nil
}

// trackChangedServices determines which services need restart based on changed artifact paths.
func (c *SyncCommand) trackChangedServices(changedPaths []string, serviceChanges map[string]platform.ChangeStatus, force bool, servicesToRestart map[string]bool) {
	changedPathSet := make(map[string]bool)
	for _, path := range changedPaths {
		changedPathSet[path] = true
	}

	for serviceName, changeStatus := range serviceChanges {
		// Check if any of this service's artifacts changed
		serviceChanged := false
		for _, artifactPath := range changeStatus.ArtifactPaths {
			if changedPathSet[artifactPath] {
				serviceChanged = true
				break
			}
		}

		if serviceChanged || force {
			servicesToRestart[serviceName] = true
		}
	}
}
