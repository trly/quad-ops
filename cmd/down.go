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

	"github.com/spf13/cobra"
)

// DownOptions holds down command options.
type DownOptions struct {
	Services []string
	All      bool
	Purge    bool
}

// DownDeps holds down dependencies.
type DownDeps struct {
	CommonDeps
	Lifecycle     LifecycleInterface
	ArtifactStore ArtifactStoreInterface
}

// DownCommand represents the down command for quad-ops CLI.
type DownCommand struct{}

// NewDownCommand creates a new DownCommand.
func NewDownCommand() *DownCommand {
	return &DownCommand{}
}

// getApp retrieves the App from the command context.
func (c *DownCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for stopping managed units.
func (c *DownCommand) GetCobraCommand() *cobra.Command {
	var opts DownOptions

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop managed services",
		Long: `Stop managed services synchronized from repositories.

By default, stops all services. Use --services to specify which services to stop.
Use --purge to delete service artifacts from disk after stopping.

Examples:
  quad-ops down                              # Stop all services
  quad-ops down --services web-service       # Stop specific service
  quad-ops down --services web,api,db        # Stop multiple services
  quad-ops down --all --purge                # Stop all and delete artifacts`,
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

	downCmd.Flags().StringSliceVarP(&opts.Services, "services", "s", nil, "Comma-separated list of services to stop")
	downCmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Explicitly stop all services (default behavior)")
	downCmd.Flags().BoolVarP(&opts.Purge, "purge", "p", false, "Delete service artifacts from disk after stopping")

	return downCmd
}

// buildDeps creates production dependencies for the down command.
// Note: Platform-specific dependency (Lifecycle) is obtained via lazy getter in Run().
func (c *DownCommand) buildDeps(app *App) DownDeps {
	return DownDeps{
		CommonDeps:    NewRootDeps(app),
		Lifecycle:     nil, // Obtained via app.GetLifecycle(ctx) in Run()
		ArtifactStore: app.ArtifactStore,
	}
}

// Run executes the down command with injected dependencies.
func (c *DownCommand) Run(ctx context.Context, app *App, opts DownOptions, deps DownDeps) error {
	// Get platform-specific component via lazy getter
	lifecycle, err := app.GetLifecycle(ctx)
	if err != nil {
		return fmt.Errorf("platform not supported: %w", err)
	}
	deps.Lifecycle = lifecycle

	// 1. Determine target services
	var servicesToStop []string

	if len(opts.Services) > 0 {
		// Use specified services
		servicesToStop = opts.Services
		deps.Logger.Debug("Stopping specified services", "services", servicesToStop)
	} else {
		// Query ArtifactStore for all services
		deps.Logger.Debug("Querying artifact store for all services")
		artifacts, err := deps.ArtifactStore.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list artifacts: %w", err)
		}

		if len(artifacts) == 0 {
			deps.Logger.Info("No managed services found")
			return nil
		}

		// Extract service names from service artifacts only (.container, .plist)
		serviceSet := make(map[string]bool)
		for _, artifact := range artifacts {
			// Only parse service artifacts - volumes and networks cannot be stopped
			if isServiceArtifact(artifact.Path) {
				serviceName := parseServiceNameFromArtifact(artifact.Path)
				if serviceName != "" {
					serviceSet[serviceName] = true
				}
			}
		}

		for serviceName := range serviceSet {
			servicesToStop = append(servicesToStop, serviceName)
		}

		deps.Logger.Debug("Found services from artifacts", "count", len(servicesToStop))
	}

	if len(servicesToStop) == 0 {
		deps.Logger.Info("No services to stop")
		return nil
	}

	// 2. Stop services using Lifecycle
	deps.Logger.Info("Stopping services", "count", len(servicesToStop))
	stopErrors := deps.Lifecycle.StopMany(ctx, servicesToStop)

	// Track failures
	failedServices := make([]string, 0)
	for serviceName, err := range stopErrors {
		if err != nil {
			deps.Logger.Error("Failed to stop service", "service", serviceName, "error", err)
			failedServices = append(failedServices, serviceName)
		} else {
			deps.Logger.Info("Service stopped", "service", serviceName)
		}
	}

	// 3. Optional: Purge artifacts if requested
	if opts.Purge {
		deps.Logger.Info("Purging service artifacts")

		// List all artifacts to find paths to delete
		artifacts, err := deps.ArtifactStore.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list artifacts for purge: %w", err)
		}

		// Build list of artifact paths to delete
		pathsToDelete := make([]string, 0)
		for _, artifact := range artifacts {
			// If --services specified, only delete artifacts for those services
			if len(opts.Services) > 0 {
				// Only match service artifacts against the services list
				// Volumes and networks are standalone resources
				if isServiceArtifact(artifact.Path) {
					serviceName := parseServiceNameFromArtifact(artifact.Path)
					if c.shouldDeleteService(serviceName, opts.Services) {
						pathsToDelete = append(pathsToDelete, artifact.Path)
					}
				}
			} else {
				// Delete all artifacts (services, volumes, networks)
				pathsToDelete = append(pathsToDelete, artifact.Path)
			}
		}

		if len(pathsToDelete) > 0 {
			deps.Logger.Debug("Deleting artifacts", "count", len(pathsToDelete))
			if err := deps.ArtifactStore.Delete(ctx, pathsToDelete); err != nil {
				return fmt.Errorf("failed to delete artifacts: %w", err)
			}

			// Reload service manager after artifact deletion
			deps.Logger.Debug("Reloading service manager after purge")
			if err := deps.Lifecycle.Reload(ctx); err != nil {
				return fmt.Errorf("failed to reload service manager: %w", err)
			}

			deps.Logger.Info("Artifacts purged", "count", len(pathsToDelete))
		}
	}

	// Report overall status
	stoppedCount := len(servicesToStop) - len(failedServices)
	if len(failedServices) > 0 {
		return fmt.Errorf("failed to stop %d of %d services", len(failedServices), len(servicesToStop))
	}

	deps.Logger.Info("Services stopped successfully", "count", stoppedCount)
	return nil
}

// shouldDeleteService checks if a service should be deleted based on the services list.
func (c *DownCommand) shouldDeleteService(serviceName string, targetServices []string) bool {
	for _, target := range targetServices {
		if serviceName == target {
			return true
		}
	}
	return false
}
