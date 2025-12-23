package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/compose"
)

type ValidateCmd struct {
	Path string `arg:"" optional:"" help:"Path to the compose file or directory"`
}

// checkSecrets validates a loaded project and reports any missing secrets.
func (v *ValidateCmd) checkSecrets(ctx context.Context, project *types.Project, filePath string) {
	missingSecrets, err := compose.CheckMissingSecrets(ctx, project)
	if err != nil {
		fmt.Printf("WARNING: %s: failed to check secrets: %v\n", filePath, err)
	}
	for _, ms := range missingSecrets {
		fmt.Printf("WARNING: %s: service %q requires missing secrets: %v\n",
			filePath, ms.ServiceName, ms.MissingSecrets)
	}
}

// processLoadedProject handles validation errors and secret checks for a loaded project
// Returns 1 if there was an error, 0 otherwise.
func (v *ValidateCmd) processLoadedProject(ctx context.Context, lp compose.LoadedProject, repoName string, _ bool) int {
	if lp.Error != nil {
		if repoName != "" {
			fmt.Printf("%s (%s): %v\n", repoName, lp.FilePath, lp.Error)
		} else {
			fmt.Printf("%s: %v\n", lp.FilePath, lp.Error)
		}
		return 1
	}

	if lp.Project != nil {
		v.checkSecrets(ctx, lp.Project, lp.FilePath)
	}
	return 0
}

func (v *ValidateCmd) Run(globals *Globals) error {
	ctx := context.Background()
	var failures int

	// If -p flag was explicitly provided, only validate the specified path
	// otherwise validate repositories from configuration
	if v.Path != "" {
		path := v.Path

		// Check if path is a directory or file
		pathInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to access path: %w", err)
		}

		pathFailures, err := func() (int, error) {
			if pathInfo.IsDir() {
				return v.validateDirectory(ctx, path, globals.Verbose)
			}
			return v.validateFile(ctx, path, globals.Verbose)
		}()
		if err != nil {
			return err
		}
		failures += pathFailures
	} else {
		// Validate repositories from configuration if available and no path specified
		if globals.AppCfg != nil && len(globals.AppCfg.Repositories) > 0 {
			repoFailures, err := v.validateRepositories(ctx, globals)
			if err != nil && globals.Verbose {
				fmt.Printf("WARNING: failed to validate repositories: %v\n", err)
			}
			failures += repoFailures
		}
	}

	if failures > 0 {
		return fmt.Errorf("%d validation error(s) found", failures)
	}

	return nil
}

func (v *ValidateCmd) validateRepositories(ctx context.Context, globals *Globals) (int, error) {
	var failures int

	for _, repo := range globals.AppCfg.Repositories {
		// Build the local path for the repository
		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)

		// Determine compose path
		scanPath := filepath.Join(repoPath, repo.ComposeDir)
		if repo.ComposeDir == "" {
			scanPath = repoPath
		}

		// Load all compose projects recursively
		projects, err := compose.LoadAll(ctx, scanPath, nil)
		if err != nil {
			if globals.Verbose {
				fmt.Printf("WARNING: %s: failed to scan: %v\n", repo.Name, err)
			}
			continue
		}

		if len(projects) == 0 {
			if globals.Verbose {
				fmt.Printf("WARNING: %s: no compose files found\n", repo.Name)
			}
			continue
		}

		// Check for validation errors in loaded projects
		for _, lp := range projects {
			failures += v.processLoadedProject(ctx, lp, repo.Name, globals.Verbose)
		}
	}

	return failures, nil
}

func (v *ValidateCmd) validateFile(ctx context.Context, path string, _ bool) (int, error) {
	result, err := compose.Load(ctx, path, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to load compose project: %w", err)
	}

	if result == nil {
		return 0, fmt.Errorf("no project loaded")
	}

	// Check for missing secrets
	v.checkSecrets(ctx, result, path)
	return 0, nil
}

func (v *ValidateCmd) validateDirectory(ctx context.Context, path string, verbose bool) (int, error) {
	results, err := compose.LoadAll(ctx, path, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no compose files found in directory")
	}

	var failures int
	for _, lp := range results {
		failures += v.processLoadedProject(ctx, lp, "", verbose)
	}

	if failures > 0 {
		return failures, fmt.Errorf("%d compose file(s) failed validation", failures)
	}

	return 0, nil
}
