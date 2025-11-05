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
	"github.com/trly/quad-ops/internal/repository"
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
func (c *SyncCommand) buildDeps(app *App) SyncDeps {
	return SyncDeps{
		CommonDeps:       NewRootDeps(app),
		GitSyncer:        app.GitSyncer,
		ComposeProcessor: app.ComposeProcessor,
		ArtifactStore:    app.ArtifactStore,
		Renderer:         nil, // Obtained via app.GetRenderer(ctx) in Run()
		Lifecycle:        nil, // Obtained via app.GetLifecycle(ctx) in Run()
	}
}

// Run executes the sync command with injected dependencies.
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

	if opts.DryRun {
		deps.Logger.Info("Dry-run mode enabled - no changes will be made")
		return nil
	}

	return c.syncRepositories(ctx, app, opts, deps)
}

// syncRepositories performs the actual repository synchronization.
func (c *SyncCommand) syncRepositories(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps) error {
	reposToSync, err := c.filterRepositories(app.Config.Repositories, opts)
	if err != nil {
		return err
	}

	results, err := deps.GitSyncer.SyncAll(ctx, reposToSync)
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	servicesToRestart := make(map[string]bool)
	anyChanges := false

	for _, result := range results {
		if err := c.handleSyncResult(ctx, app, opts, deps, result, servicesToRestart, &anyChanges); err != nil {
			deps.Logger.Error("Failed to process repository", "repo", result.Repository.Name, "error", err)
		}
	}

	if anyChanges || opts.Force {
		deps.Logger.Info("Reloading service manager")
		if err := deps.Lifecycle.Reload(ctx); err != nil {
			return fmt.Errorf("failed to reload service manager: %w", err)
		}

		if len(servicesToRestart) > 0 {
			names := c.sortedServiceNames(servicesToRestart)
			restartErrs := deps.Lifecycle.RestartMany(ctx, names)
			for name, rerr := range restartErrs {
				if rerr != nil {
					deps.Logger.Error("Failed to restart service", "service", name, "error", rerr)
				} else {
					deps.Logger.Info("Service restarted", "service", name)
				}
			}
		}
	} else {
		deps.Logger.Info("No changes detected")
	}

	return nil
}

// filterRepositories filters repositories based on sync options.
func (c *SyncCommand) filterRepositories(repos []config.Repository, opts SyncOptions) ([]config.Repository, error) {
	if opts.RepoName == "" {
		return repos, nil
	}

	for _, repo := range repos {
		if repo.Name == opts.RepoName {
			return []config.Repository{repo}, nil
		}
	}

	return nil, fmt.Errorf("repository not found: %s", opts.RepoName)
}

// handleSyncResult processes a single repository sync result.
func (c *SyncCommand) handleSyncResult(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps, result repository.SyncResult, servicesToRestart map[string]bool, anyChanges *bool) error {
	if result.Error != nil {
		return result.Error
	}

	if !result.Changed && !opts.Force {
		deps.Logger.Debug("Repository unchanged, skipping", "repo", result.Repository.Name)
		return nil
	}

	repoPath := filepath.Join(app.Config.RepositoryDir, result.Repository.Name)
	projects, err := compose.ReadProjects(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read compose projects: %w", err)
	}

	if len(projects) == 0 {
		deps.Logger.Debug("No compose projects found", "repo", result.Repository.Name)
		return nil
	}

	for _, project := range projects {
		specs, err := deps.ComposeProcessor.Process(ctx, project)
		if err != nil {
			deps.Logger.Error("Failed to process compose project", "repo", result.Repository.Name, "project", project.Name, "error", err)
			continue
		}

		deps.Logger.Debug("Processed compose project", "repo", result.Repository.Name, "project", project.Name, "services", len(specs))

		renderResult, err := deps.Renderer.Render(ctx, specs)
		if err != nil {
			deps.Logger.Error("Failed to render artifacts", "repo", result.Repository.Name, "project", project.Name, "error", err)
			continue
		}

		changedPaths, err := deps.ArtifactStore.Write(ctx, renderResult.Artifacts)
		if err != nil {
			deps.Logger.Error("Failed to write artifacts", "repo", result.Repository.Name, "project", project.Name, "error", err)
			continue
		}

		if len(changedPaths) > 0 {
			*anyChanges = true
			c.trackChangedServices(changedPaths, renderResult.ServiceChanges, opts.Force, servicesToRestart)
			deps.Logger.Info("Artifacts written", "repo", result.Repository.Name, "project", project.Name, "changed", len(changedPaths))
		} else {
			deps.Logger.Debug("No artifact changes", "repo", result.Repository.Name, "project", project.Name)
		}
	}

	return nil
}

// trackChangedServices marks services for restart based on changed artifact paths.
func (c *SyncCommand) trackChangedServices(changedPaths []string, serviceChanges map[string]platform.ChangeStatus, force bool, servicesToRestart map[string]bool) {
	if force {
		for serviceName := range serviceChanges {
			servicesToRestart[serviceName] = true
		}
		return
	}

	changedPathSet := make(map[string]bool)
	for _, path := range changedPaths {
		changedPathSet[path] = true
	}

	for serviceName, changeStatus := range serviceChanges {
		for _, artifactPath := range changeStatus.ArtifactPaths {
			if changedPathSet[artifactPath] {
				servicesToRestart[serviceName] = true
				break
			}
		}
	}
}

// sortedServiceNames returns a sorted slice of service names from the map.
func (c *SyncCommand) sortedServiceNames(services map[string]bool) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	return names
}
