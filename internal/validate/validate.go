// Package validate provides functions to validate various aspects of the application.
package validate

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/trly/quad-ops/internal/log"
)

// CommandRunner defines an interface for executing commands.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct {
	logger log.Logger
}

// NewRealCommandRunner creates a new RealCommandRunner with the provided logger.
func NewRealCommandRunner(logger log.Logger) *RealCommandRunner {
	return &RealCommandRunner{
		logger: logger,
	}
}

// Run executes a command and returns its output.
// WARNING: This method executes arbitrary commands and should only be used with trusted input.
// Callers must validate command names and arguments to prevent command injection.
func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	// Basic validation - reject empty command names
	if name == "" {
		return nil, fmt.Errorf("command name cannot be empty")
	}

	// Log command execution for security auditing
	r.logger.Debug("Executing command", "name", name, "args", args)

	return exec.Command(name, args...).Output()
}

// Validator provides system requirements validation with dependency injection.
type Validator struct {
	logger log.Logger
	runner CommandRunner
}

// NewValidator creates a new Validator with the provided logger and command runner.
func NewValidator(logger log.Logger, runner CommandRunner) *Validator {
	return &Validator{
		logger: logger,
		runner: runner,
	}
}

// NewValidatorWithDefaults creates a new Validator with default dependencies.
func NewValidatorWithDefaults(logger log.Logger) *Validator {
	return &Validator{
		logger: logger,
		runner: NewRealCommandRunner(logger),
	}
}

// SystemRequirements checks if all required system tools are installed.
func (v *Validator) SystemRequirements() error {
	v.logger.Debug("Validating systemd availability")

	systemdVersion, err := v.runner.Run("systemctl", "--version")
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	v.logger.Debug("Validating podman availability")

	_, err = v.runner.Run("podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	v.logger.Debug("Validating podman-system-generator availability")

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = v.runner.Run("test", "-f", generatorPath)
	if err != nil {
		return fmt.Errorf("podman systemd generator not found (ensure podman is properly installed)")
	}

	return nil
}

// SystemRequirements checks if all required system tools are installed.
// Deprecated: Use NewValidator and Validator.SystemRequirements instead.
func SystemRequirements() error {
	logger := log.NewLogger(false)
	validator := NewValidatorWithDefaults(logger)
	return validator.SystemRequirements()
}
