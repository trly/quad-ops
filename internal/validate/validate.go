// Package validate provides functions to validate various aspects of the application.
package validate

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/trly/quad-ops/internal/log"
)

// CommandRunner defines an interface for executing commands.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

// Run executes a command and returns its output.
// WARNING: This method executes arbitrary commands and should only be used with trusted input.
// Callers must validate command names and arguments to prevent command injection.
func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	// Basic validation - reject empty command names
	if name == "" {
		return nil, fmt.Errorf("command name cannot be empty")
	}

	// Log command execution for security auditing
	log.GetLogger().Debug("Executing command", "name", name, "args", args)

	return exec.Command(name, args...).Output()
}

// default runner for use in production code.
var defaultRunner CommandRunner = &RealCommandRunner{}

// SetCommandRunner allows tests to inject a mock runner.
func SetCommandRunner(runner CommandRunner) {
	defaultRunner = runner
}

// ResetCommandRunner restores the default runner.
func ResetCommandRunner() {
	defaultRunner = &RealCommandRunner{}
}

// SystemRequirements checks if all required system tools are installed.
func SystemRequirements() error {
	log.GetLogger().Debug("Validating systemd availability")

	systemdVersion, err := defaultRunner.Run("systemctl", "--version")
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	log.GetLogger().Debug("Validating podman availability")

	_, err = defaultRunner.Run("podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	log.GetLogger().Debug("Validating podman-system-generator availability")

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = defaultRunner.Run("test", "-f", generatorPath)
	if err != nil {
		return fmt.Errorf("podman systemd generator not found (ensure podman is properly installed)")
	}

	return nil
}

// ValidatePath validates that a path doesn't contain path traversal sequences.
// It uses filepath.Clean to normalize the path and checks for traversal attempts.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to normalize it and resolve any traversal sequences
	cleanPath := filepath.Clean(path)

	// If the cleaned path is different and contains traversal, it's suspicious
	if cleanPath != path && strings.Contains(path, "..") {
		return fmt.Errorf("path contains path traversal sequence")
	}

	// Check if the cleaned path tries to go above the current directory for relative paths
	if !filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path attempts to traverse above working directory")
	}

	return nil
}

// ValidatePathWithinBase ensures a path stays within a base directory after cleaning.
// This is more secure than ValidatePath alone for critical file operations.
func ValidatePathWithinBase(path, basePath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	// Clean both paths to normalize them
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(basePath)

	// Make paths absolute for proper comparison
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(absBase, cleanPath)
	}

	// Clean the final path
	absPath = filepath.Clean(absPath)

	// Ensure the final path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path escapes base directory")
	}

	return absPath, nil
}
