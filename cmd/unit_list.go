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

// Package cmd provides unit command functionality for quad-ops CLI
package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
)

// ListOptions holds list command options.
type ListOptions struct {
	Status        bool
	UseFilesystem bool // If true, use filesystem artifacts instead of repo artifacts
}

// ListDeps holds list dependencies.
type ListDeps struct {
	CommonDeps
	RepoArtifactStore repository.ArtifactStore // Git-managed artifacts
	ArtifactStore     repository.ArtifactStore // Deployed filesystem artifacts
}

// ListCommand represents the unit list command.
type ListCommand struct{}

// NewListCommand creates a new ListCommand.
func NewListCommand() *ListCommand {
	return &ListCommand{}
}

// getApp retrieves the App from the command context.
func (c *ListCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for listing units.
func (c *ListCommand) GetCobraCommand() *cobra.Command {
	var opts ListOptions

	unitListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists deployed artifacts currently managed by quad-ops",
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

	unitListCmd.Flags().BoolVarP(&opts.Status, "status", "s", false, "Include service status information")
	unitListCmd.Flags().BoolVar(&opts.UseFilesystem, "use-fs-artifacts", false, "Use filesystem artifacts instead of git-managed artifacts (Linux only)")

	return unitListCmd
}

// Run executes the list command with injected dependencies.
func (c *ListCommand) Run(ctx context.Context, app *App, opts ListOptions, deps ListDeps) error {
	var artifacts []platform.Artifact
	var err error

	// Determine which artifacts to display based on flags
	if opts.UseFilesystem {
		// Use filesystem artifacts (deployed in QuadletDir)
		deps.Logger.Debug("Listing artifacts from filesystem")
		artifacts, err = deps.ArtifactStore.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list filesystem artifacts: %w", err)
		}
		artifacts = filterArtifactsForPlatform(artifacts, app.Config)
	} else {
		// Default: use git-managed artifacts from repository
		deps.Logger.Debug("Listing artifacts from repository")
		artifacts, err = deps.RepoArtifactStore.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list repository artifacts: %w", err)
		}
		artifacts = filterArtifactsForPlatform(artifacts, app.Config)
	}

	// For repo artifacts, filter to only those that are actually deployed
	var deployedArtifacts []platform.Artifact
	if !opts.UseFilesystem {
		for _, artifact := range artifacts {
			deployedPath := filepath.Join(app.Config.QuadletDir, artifact.Path)
			if _, err := deps.FileSystem.Stat(deployedPath); err == nil {
				deployedArtifacts = append(deployedArtifacts, artifact)
			}
		}
	} else {
		// Filesystem artifacts are already deployed by definition
		deployedArtifacts = artifacts
	}

	// Get lifecycle if status is requested
	var lifecycle platform.Lifecycle
	if opts.Status {
		lc, err := app.GetLifecycle(ctx)
		if err != nil {
			return fmt.Errorf("failed to get lifecycle: %w", err)
		}
		lifecycle = lc
	}

	if len(deployedArtifacts) == 0 {
		deps.Logger.Info("No deployed artifacts found")
		return nil
	}

	// Setup table with appropriate columns
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	var tbl table.Table
	if opts.Status {
		tbl = table.New("Path", "Type", "Hash", "Active", "State")
	} else {
		tbl = table.New("Path", "Type", "Hash")
	}
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	// Display artifacts
	for _, artifact := range deployedArtifacts {
		hashStr := artifact.Hash
		if len(hashStr) > 12 {
			hashStr = hashStr[:12] // First 12 chars
		}

		artifactType := extractArtifactType(artifact.Path)

		if opts.Status && isServiceArtifact(artifact.Path) {
			// Fetch status for service artifacts (.container on systemd, .plist on launchd)
			serviceName := parseServiceNameFromArtifact(artifact.Path)
			status, err := lifecycle.Status(ctx, serviceName)
			if err != nil {
				deps.Logger.Debug("Error getting service status", "service", serviceName, "error", err)
				tbl.AddRow(artifact.Path, artifactType, hashStr, "UNKNOWN", "-")
			} else {
				activeState := "inactive"
				if status.Active {
					activeState = "active"
				}
				tbl.AddRow(artifact.Path, artifactType, hashStr, activeState, status.State)
			}
		} else {
			tbl.AddRow(artifact.Path, artifactType, hashStr)
		}
	}

	tbl.Print()
	return nil
}

// buildDeps creates production dependencies for the list command.
// Note: Lifecycle is obtained via lazy getter in Run() when status is requested.
func (c *ListCommand) buildDeps(app *App) ListDeps {
	return ListDeps{
		CommonDeps:        NewRootDeps(app),
		RepoArtifactStore: app.RepoArtifactStore,
		ArtifactStore:     app.ArtifactStore,
	}
}

// extractArtifactType extracts the type from an artifact path.
// E.g., "myservice.container" -> "container", "com.example.svc.plist" -> "plist".
func extractArtifactType(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimPrefix(ext, ".")
}
