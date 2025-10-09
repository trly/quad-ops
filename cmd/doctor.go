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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
)

// DoctorOptions holds doctor command options.
type DoctorOptions struct {
	// Currently no specific options for doctor command
}

// DoctorDeps holds doctor dependencies.
type DoctorDeps struct {
	CommonDeps
	NewGitRepo      func(config.Repository, config.Provider) *git.Repository
	ViperConfigFile func() string
	GetOS           func() string
}

// DoctorCommand represents the doctor command for quad-ops CLI.
type DoctorCommand struct{}

// NewDoctorCommand creates a new DoctorCommand.
func NewDoctorCommand() *DoctorCommand {
	return &DoctorCommand{}
}

// getApp retrieves the App from the command context.
func (c *DoctorCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// CheckResult represents the result of a diagnostic check.
type CheckResult struct {
	Name        string
	Passed      bool
	Message     string
	Suggestions []string
}

// GetCobraCommand returns the cobra command for doctor operations.
func (c *DoctorCommand) GetCobraCommand() *cobra.Command {
	var opts DoctorOptions

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and configuration",
		Long: `Check system health and configuration for quad-ops.

The doctor command performs comprehensive checks of:
- System requirements (systemd/podman on Linux, launchd/podman on macOS)
- Configuration file validity
- Directory permissions and accessibility
- Repository connectivity
- File system requirements

This helps diagnose common setup and configuration issues.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return doctorCmd
}

// buildDeps creates production dependencies for the doctor command.
func (c *DoctorCommand) buildDeps(app *App) DoctorDeps {
	return DoctorDeps{
		CommonDeps:      NewRootDeps(app),
		NewGitRepo:      git.NewGitRepository,
		ViperConfigFile: func() string { return viper.GetViper().ConfigFileUsed() },
		GetOS:           func() string { return runtime.GOOS },
	}
}

// Run executes the doctor command with injected dependencies.
func (c *DoctorCommand) Run(_ context.Context, app *App, _ DoctorOptions, deps DoctorDeps) error {
	// Collect all diagnostic results
	var results []CheckResult
	var failureCount int

	// Run all checks
	results = append(results, c.checkSystemRequirements(app, deps)...)
	results = append(results, c.checkConfiguration(app, deps)...)
	results = append(results, c.checkDirectories(app, deps)...)
	results = append(results, c.checkRepositories(app, deps)...)

	// Count failures
	for _, result := range results {
		if !result.Passed {
			failureCount++
		}
	}

	// Display results based on output format
	if app.OutputFormat == "text" {
		// Traditional text output
		if app.Config.Verbose {
			c.displayDetailedResults(results)
		} else {
			c.displaySummaryResults(results)
		}

		// Return error instead of exiting
		if failureCount > 0 {
			if !app.Config.Verbose {
				fmt.Printf("\n%d checks failed. Run with --verbose for details.\n", failureCount)
			}
			return fmt.Errorf("doctor found %d issues", failureCount)
		} else if app.Config.Verbose {
			fmt.Println("\n✓ All checks passed")
		}
	} else {
		// Structured output (JSON/YAML)
		c.outputStructuredResults(app, results, failureCount)
		if failureCount > 0 {
			return fmt.Errorf("doctor found %d issues", failureCount)
		}
	}

	return nil
}

// checkSystemRequirements validates core system dependencies.
func (c *DoctorCommand) checkSystemRequirements(app *App, deps DoctorDeps) []CheckResult {
	var results []CheckResult

	// Check platform-specific requirements
	err := app.Validator.SystemRequirements()
	if err != nil {
		// Platform-specific suggestions
		var suggestions []string
		platform := deps.GetOS()

		switch platform {
		case "linux":
			suggestions = []string{
				"Install systemd if running on a systemd-based system",
				"Install podman for container operations",
				"Ensure systemd and podman are in your PATH",
			}
		case "darwin":
			suggestions = []string{
				"Install podman via Podman Desktop (https://podman-desktop.io) or Homebrew (brew install podman)",
				"Ensure podman is in your PATH",
				"launchd is built-in on macOS and should be available by default",
			}
		default:
			suggestions = []string{
				"quad-ops requires Linux (systemd) or macOS (launchd) for service management",
			}
		}

		results = append(results, CheckResult{
			Name:        "System Requirements",
			Passed:      false,
			Message:     err.Error(),
			Suggestions: suggestions,
		})
	} else {
		// Platform-specific success message
		var message string
		platform := deps.GetOS()

		switch platform {
		case "linux":
			message = "systemd and podman are available"
		case "darwin":
			message = "launchd and podman are available"
		default:
			message = "platform requirements met"
		}

		results = append(results, CheckResult{
			Name:    "System Requirements",
			Passed:  true,
			Message: message,
		})
	}

	return results
}

// checkConfiguration validates configuration file and settings.
func (c *DoctorCommand) checkConfiguration(app *App, deps DoctorDeps) []CheckResult {
	var results []CheckResult

	// Check if config file exists and is readable
	configFile := deps.ViperConfigFile()
	if configFile == "" {
		results = append(results, CheckResult{
			Name:    "Configuration File",
			Passed:  false,
			Message: "No configuration file found",
			Suggestions: []string{
				"Create a configuration file at ~/.config/quad-ops/config.yaml",
				"Or specify config file path with --config flag",
				"Run 'quad-ops config' to see current configuration",
			},
		})
	} else {
		if _, err := deps.FileSystem.Stat(configFile); err != nil {
			results = append(results, CheckResult{
				Name:    "Configuration File",
				Passed:  false,
				Message: fmt.Sprintf("Configuration file not accessible: %v", err),
				Suggestions: []string{
					"Check file permissions on " + configFile,
					"Verify the file path is correct",
				},
			})
		} else {
			results = append(results, CheckResult{
				Name:    "Configuration File",
				Passed:  true,
				Message: fmt.Sprintf("Configuration loaded from %s", configFile),
			})
		}
	}

	// Check if repositories are configured
	if len(app.Config.Repositories) == 0 {
		results = append(results, CheckResult{
			Name:    "Repository Configuration",
			Passed:  false,
			Message: "No repositories configured",
			Suggestions: []string{
				"Add repository configurations to your config file",
				"Each repository should specify name, url, and target branch",
			},
		})
	} else {
		results = append(results, CheckResult{
			Name:    "Repository Configuration",
			Passed:  true,
			Message: fmt.Sprintf("%d repositories configured", len(app.Config.Repositories)),
		})
	}

	return results
}

// checkDirectories validates directory permissions and accessibility.
func (c *DoctorCommand) checkDirectories(app *App, deps DoctorDeps) []CheckResult {
	var results []CheckResult

	// Check quadlet directory
	quadletDir := app.Config.QuadletDir
	if err := c.checkDirectory("Quadlet Directory", quadletDir, deps); err != nil {
		suggestions := []string{
			fmt.Sprintf("Create directory: mkdir -p %s", quadletDir),
			fmt.Sprintf("Fix permissions: chmod 755 %s", quadletDir),
		}
		results = append(results, CheckResult{
			Name:        "Quadlet Directory",
			Passed:      false,
			Message:     err.Error(),
			Suggestions: suggestions,
		})
	} else {
		results = append(results, CheckResult{
			Name:    "Quadlet Directory",
			Passed:  true,
			Message: fmt.Sprintf("Directory accessible at %s", quadletDir),
		})
	}

	// Check repository directory
	repoDir := app.Config.RepositoryDir
	if err := c.checkDirectory("Repository Directory", repoDir, deps); err != nil {
		suggestions := []string{
			fmt.Sprintf("Create directory: mkdir -p %s", repoDir),
			fmt.Sprintf("Fix permissions: chmod 755 %s", repoDir),
		}
		results = append(results, CheckResult{
			Name:        "Repository Directory",
			Passed:      false,
			Message:     err.Error(),
			Suggestions: suggestions,
		})
	} else {
		results = append(results, CheckResult{
			Name:    "Repository Directory",
			Passed:  true,
			Message: fmt.Sprintf("Directory accessible at %s", repoDir),
		})
	}

	return results
}

// checkRepositories validates repository connectivity and accessibility.
func (c *DoctorCommand) checkRepositories(app *App, deps DoctorDeps) []CheckResult {
	results := make([]CheckResult, 0, len(app.Config.Repositories))

	for _, repoConfig := range app.Config.Repositories {
		gitRepo := deps.NewGitRepo(repoConfig, app.ConfigProvider)

		// Check if repository directory exists
		repoPath := gitRepo.Path
		if _, err := deps.FileSystem.Stat(repoPath); err != nil {
			suggestions := []string{
				"Run 'quad-ops sync' to clone repositories",
				"Check network connectivity to repository URL",
				"Verify git credentials if using private repositories",
			}
			results = append(results, CheckResult{
				Name:        fmt.Sprintf("Repository: %s", repoConfig.Name),
				Passed:      false,
				Message:     fmt.Sprintf("Repository not cloned locally: %v", err),
				Suggestions: suggestions,
			})
			continue
		}

		// Check if it's a valid git repository
		if !c.isValidGitRepo(repoPath, deps) {
			suggestions := []string{
				fmt.Sprintf("Remove invalid directory: rm -rf %s", repoPath),
				"Run 'quad-ops sync' to re-clone repository",
			}
			results = append(results, CheckResult{
				Name:        fmt.Sprintf("Repository: %s", repoConfig.Name),
				Passed:      false,
				Message:     "Directory exists but is not a valid git repository",
				Suggestions: suggestions,
			})
			continue
		}

		// Check compose directory if specified
		if repoConfig.ComposeDir != "" {
			composeDir := filepath.Join(repoPath, repoConfig.ComposeDir)
			if _, err := deps.FileSystem.Stat(composeDir); err != nil {
				suggestions := []string{
					fmt.Sprintf("Verify compose directory path in configuration: %s", repoConfig.ComposeDir),
					"Check if the directory exists in the repository",
				}
				results = append(results, CheckResult{
					Name:        fmt.Sprintf("Repository: %s", repoConfig.Name),
					Passed:      false,
					Message:     fmt.Sprintf("Compose directory not found: %s", repoConfig.ComposeDir),
					Suggestions: suggestions,
				})
				continue
			}
		}

		results = append(results, CheckResult{
			Name:    fmt.Sprintf("Repository: %s", repoConfig.Name),
			Passed:  true,
			Message: fmt.Sprintf("Repository accessible at %s", repoPath),
		})
	}

	return results
}

// checkDirectory validates a directory exists and is accessible.
func (c *DoctorCommand) checkDirectory(_, path string, deps DoctorDeps) error {
	if path == "" {
		return fmt.Errorf("directory path is empty")
	}

	// Check if directory exists
	stat, err := deps.FileSystem.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return fmt.Errorf("cannot access directory: %v", err)
	}

	// Check if it's actually a directory
	if !stat.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}

	// Check if directory is writable
	testFile := filepath.Join(path, ".quad-ops-test")
	if err := deps.FileSystem.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return fmt.Errorf("directory is not writable: %v", err)
	}
	if err := deps.FileSystem.Remove(testFile); err != nil {
		deps.Logger.Debug("Failed to cleanup test file", "file", testFile, "error", err)
	}

	return nil
}

// isValidGitRepo checks if the given path contains a valid git repository.
func (c *DoctorCommand) isValidGitRepo(path string, deps DoctorDeps) bool {
	gitDir := filepath.Join(path, ".git")
	if stat, err := deps.FileSystem.Stat(gitDir); err != nil || !stat.IsDir() {
		return false
	}
	return true
}

// displaySummaryResults shows a brief summary of check results.
func (c *DoctorCommand) displaySummaryResults(results []CheckResult) {
	var failed []CheckResult

	for _, result := range results {
		if !result.Passed {
			failed = append(failed, result)
		}
	}

	if len(failed) > 0 {
		fmt.Println("Issues found:")
		for _, result := range failed {
			fmt.Printf("✗ %s: %s\n", result.Name, result.Message)
		}
	}
}

// displayDetailedResults shows detailed information about all checks.
func (c *DoctorCommand) displayDetailedResults(results []CheckResult) {
	fmt.Println("System Health Check Results:")
	fmt.Println(strings.Repeat("=", 40))

	for _, result := range results {
		if result.Passed {
			fmt.Printf("✓ %s: %s\n", result.Name, result.Message)
		} else {
			fmt.Printf("✗ %s: %s\n", result.Name, result.Message)
			if len(result.Suggestions) > 0 {
				fmt.Println("  Suggestions:")
				for _, suggestion := range result.Suggestions {
					fmt.Printf("    - %s\n", suggestion)
				}
			}
		}
		fmt.Println()
	}
}

// outputStructuredResults outputs health check results in structured format (JSON/YAML).
func (c *DoctorCommand) outputStructuredResults(app *App, results []CheckResult, failureCount int) {
	checks := make([]CheckResultStructured, 0, len(results))
	passedCount := 0

	for _, result := range results {
		status := "failed"
		if result.Passed {
			status = "passed"
			passedCount++
		}

		checks = append(checks, CheckResultStructured{
			Name:        result.Name,
			Status:      status,
			Message:     result.Message,
			Suggestions: result.Suggestions,
		})
	}

	overall := "passed"
	if failureCount > 0 {
		overall = "failed"
	}

	output := HealthCheckOutput{
		Overall: overall,
		Checks:  checks,
		Summary: map[string]int{
			"total":  len(results),
			"passed": passedCount,
			"failed": failureCount,
		},
	}

	// Print structured output
	_ = PrintOutput(app.OutputFormat, output)
}
