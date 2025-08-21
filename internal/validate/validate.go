// Package validate provides functions to validate various aspects of the application.
package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
)

// Validator provides system requirements validation with dependency injection.
type Validator struct {
	logger log.Logger
	runner execx.Runner
}

// NewValidator creates a new Validator with the provided logger and command runner.
func NewValidator(logger log.Logger, runner execx.Runner) *Validator {
	return &Validator{
		logger: logger,
		runner: runner,
	}
}

// NewValidatorWithDefaults creates a new Validator with default dependencies.
func NewValidatorWithDefaults(logger log.Logger) *Validator {
	return &Validator{
		logger: logger,
		runner: execx.NewRealRunner(),
	}
}

// SystemRequirements checks if all required system tools are installed.
func (v *Validator) SystemRequirements() error {
	v.logger.Debug("Validating systemd availability")

	ctx := context.Background()
	systemdVersion, err := v.runner.CombinedOutput(ctx, "systemctl", "--version")
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	v.logger.Debug("Validating podman availability")

	_, err = v.runner.CombinedOutput(ctx, "podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	v.logger.Debug("Validating podman-system-generator availability")

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = v.runner.CombinedOutput(ctx, "test", "-f", generatorPath)
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
