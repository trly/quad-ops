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
)

// UpOptions holds up command options.
type UpOptions struct {
	Services []string
	Force    bool
	DryRun   bool
	RepoName string
}

// UpDeps holds up dependencies.
type UpDeps struct {
	CommonDeps
	ComposeProcessor ComposeProcessorInterface
	Renderer         RendererInterface
	ArtifactStore    ArtifactStoreInterface
	Lifecycle        LifecycleInterface
}

// UpCommand represents the up command for quad-ops CLI.
type UpCommand struct{}

// NewUpCommand creates a new UpCommand.
func NewUpCommand() *UpCommand {
	return &UpCommand{}
}

// getApp retrieves the App from the command context.
func (c *UpCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for starting managed units.
func (c *UpCommand) GetCobraCommand() *cobra.Command {
	var opts UpOptions

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Process compose files and start services",
		Long: `Process Docker Compose files from repositories and start services.

This command orchestrates the full workflow:
1. Process compose files from selected repositories
2. Render service specifications to platform artifacts
3. Write artifacts to disk (with change detection)
4. Reload service manager if changes detected
5. Start the specified services (or all if none specified)

Examples:
  quad-ops up                           # Start all services
  quad-ops up --services web,api        # Start specific services
  quad-ops up --repo my-repo            # Process only one repository
  quad-ops up --dry-run                 # Show what would be done
  quad-ops up --force                   # Force processing even without changes`,
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

	upCmd.Flags().StringSliceVar(&opts.Services, "services", nil, "Comma-separated list of services to start")
	upCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force processing even if no changes detected")
	upCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "Show what would be done without making changes")
	upCmd.Flags().StringVarP(&opts.RepoName, "repo", "r", "", "Process only a specific repository")

	return upCmd
}

// buildDeps creates production dependencies for the up command.
// Note: Platform-specific dependencies (Renderer, Lifecycle) are obtained via lazy getters in Run().
func (c *UpCommand) buildDeps(app *App) UpDeps {
	return UpDeps{
		CommonDeps:       NewRootDeps(app),
		ComposeProcessor: app.ComposeProcessor,
		Renderer:         nil, // Obtained via app.GetRenderer(ctx) in Run()
		ArtifactStore:    app.ArtifactStore,
		Lifecycle:        nil, // Obtained via app.GetLifecycle(ctx) in Run()
	}
}

// Run executes the up command with injected dependencies.
// This method orchestrates the workflow: ComposeProcessor → Renderer → ArtifactStore → Lifecycle.
//
//nolint:gocyclo // Orchestration logic requires sequential steps
func (c *UpCommand) Run(ctx context.Context, app *App, opts UpOptions, deps UpDeps) error {
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

	// Filter repositories if specific repo requested
	reposToProcess := app.Config.Repositories
	if opts.RepoName != "" {
		reposToProcess = make([]config.Repository, 0, 1)
		for _, repo := range app.Config.Repositories {
			if repo.Name == opts.RepoName {
				reposToProcess = append(reposToProcess, repo)
				break
			}
		}
		if len(reposToProcess) == 0 {
			return fmt.Errorf("repository not found: %s", opts.RepoName)
		}
	}

	// Track all available services from compose processing
	allServices := make(map[string]bool)
	anyChanges := false

	// 1. Process compose files from selected repositories
	for _, repo := range reposToProcess {
		deps.Logger.Debug("Processing repository", "repo", repo.Name)

		repoPath := filepath.Join(app.Config.RepositoryDir, repo.Name)

		// Read compose files from repository
		projects, err := compose.ReadProjects(repoPath)
		if err != nil {
			deps.Logger.Error("Failed to read compose projects", "repo", repo.Name, "error", err)
			continue
		}

		if len(projects) == 0 {
			deps.Logger.Debug("No compose projects found", "repo", repo.Name)
			continue
		}

		// Process all compose projects to service specs
		for _, project := range projects {
			specs, err := deps.ComposeProcessor.Process(ctx, project)
			if err != nil {
				deps.Logger.Error("Failed to process compose project",
					"repo", repo.Name, "project", project.Name, "error", err)
				continue
			}

			deps.Logger.Debug("Processed compose project",
				"repo", repo.Name, "project", project.Name, "services", len(specs))

			// Track service names
			for _, spec := range specs {
				allServices[spec.Name] = true
			}

			// 2. Render to platform-specific artifacts
			renderResult, err := deps.Renderer.Render(ctx, specs)
			if err != nil {
				deps.Logger.Error("Failed to render artifacts",
					"repo", repo.Name, "project", project.Name, "error", err)
				continue
			}

			// Handle dry-run mode
			if opts.DryRun {
				deps.Logger.Info("Would write artifacts (dry-run)",
					"repo", repo.Name, "project", project.Name, "count", len(renderResult.Artifacts))
				for _, artifact := range renderResult.Artifacts {
					deps.Logger.Info("  Artifact", "path", artifact.Path)
				}
				continue
			}

			// 3. Write artifacts to disk (with change detection)
			changedPaths, err := deps.ArtifactStore.Write(ctx, renderResult.Artifacts)
			if err != nil {
				deps.Logger.Error("Failed to write artifacts",
					"repo", repo.Name, "project", project.Name, "error", err)
				continue
			}

			if len(changedPaths) > 0 {
				anyChanges = true
				deps.Logger.Info("Artifacts written",
					"repo", repo.Name, "project", project.Name, "changed", len(changedPaths))
			} else {
				deps.Logger.Debug("No artifact changes", "repo", repo.Name, "project", project.Name)
			}
		}
	}

	// Handle dry-run mode early exit
	if opts.DryRun {
		serviceList := make([]string, 0, len(allServices))
		for svc := range allServices {
			serviceList = append(serviceList, svc)
		}
		deps.Logger.Info("Would start services (dry-run)", "services", serviceList)
		return nil
	}

	// 4. Reload service manager if changes detected or forced
	if anyChanges || opts.Force {
		deps.Logger.Info("Reloading service manager")
		if err := deps.Lifecycle.Reload(ctx); err != nil {
			return fmt.Errorf("failed to reload service manager: %w", err)
		}
	}

	// 5. Determine target services (filter by --services flag or all specs)
	var servicesToStart []string
	if len(opts.Services) > 0 {
		// Use specified services
		servicesToStart = opts.Services
		// Validate requested services exist
		for _, svc := range servicesToStart {
			if !allServices[svc] {
				deps.Logger.Warn("Requested service not found in compose files", "service", svc)
			}
		}
	} else {
		// Start all discovered services
		servicesToStart = make([]string, 0, len(allServices))
		for svc := range allServices {
			servicesToStart = append(servicesToStart, svc)
		}
	}

	if len(servicesToStart) == 0 {
		deps.Logger.Info("No services to start")
		return nil
	}

	// 6. Start services using Lifecycle.StartMany
	deps.Logger.Info("Starting services", "count", len(servicesToStart))

	startErrors := deps.Lifecycle.StartMany(ctx, servicesToStart)

	// Log results
	successCount := 0
	failCount := 0
	for serviceName, err := range startErrors {
		if err != nil {
			deps.Logger.Error("Failed to start service", "service", serviceName, "error", err)
			failCount++
		} else {
			deps.Logger.Info("Service started", "service", serviceName)
			successCount++
		}
	}

	if failCount > 0 {
		return fmt.Errorf("failed to start %d services", failCount)
	}

	if app.Config.Verbose {
		fmt.Printf("Successfully started %d services\n", successCount)
	}

	return nil
}
