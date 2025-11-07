// Package validate provides functions to validate various aspects of the application.
package validate

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
)

// Validator provides system requirements validation with dependency injection.
type Validator struct {
	logger          log.Logger
	runner          execx.Runner
	osGetter        func() string // For testing, defaults to runtime.GOOS
	SecretValidator *SecretValidator
}

// NewValidator creates a new Validator with the provided logger and command runner.
func NewValidator(logger log.Logger, runner execx.Runner) *Validator {
	return &Validator{
		logger:          logger,
		runner:          runner,
		osGetter:        func() string { return runtime.GOOS },
		SecretValidator: NewSecretValidator(logger),
	}
}

// WithOSGetter sets a custom OS getter for testing.
func (v *Validator) WithOSGetter(osGetter func() string) *Validator {
	v.osGetter = osGetter
	return v
}

// NewValidatorWithDefaults creates a new Validator with default dependencies.
func NewValidatorWithDefaults(logger log.Logger) *Validator {
	return &Validator{
		logger:          logger,
		runner:          execx.NewRealRunner(),
		osGetter:        func() string { return runtime.GOOS },
		SecretValidator: NewSecretValidator(logger),
	}
}

// SystemRequirements checks if all required system tools are installed.
func (v *Validator) SystemRequirements() error {
	ctx := context.Background()
	goos := v.osGetter()

	switch goos {
	case "linux":
		return v.validateLinux(ctx)
	case "darwin":
		return v.validateDarwin(ctx)
	default:
		return fmt.Errorf("unsupported platform: %s (quad-ops requires Linux with systemd or macOS with launchd)", goos)
	}
}

// validateLinux checks Linux-specific requirements (systemd + podman).
func (v *Validator) validateLinux(ctx context.Context) error {
	v.logger.Debug("Validating systemd availability")

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

// validateDarwin checks macOS-specific requirements (launchd + podman).
func (v *Validator) validateDarwin(ctx context.Context) error {
	v.logger.Debug("Validating launchd availability")

	_, err := v.runner.CombinedOutput(ctx, "launchctl", "version")
	if err != nil {
		return fmt.Errorf("launchd not available: %w", err)
	}

	v.logger.Debug("Validating podman availability")

	_, err = v.runner.CombinedOutput(ctx, "podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found (install via Podman Desktop or Homebrew): %w", err)
	}

	return nil
}

// ValidatePodmanSecretExists checks if a podman secret exists on the system.
func (v *Validator) ValidatePodmanSecretExists(ctx context.Context, secretName string) error {
	output, err := v.runner.CombinedOutput(ctx, "podman", "secret", "ls", "--format", "table {{.Name}}")
	if err != nil {
		return fmt.Errorf("failed to list podman secrets: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == secretName || strings.HasSuffix(line, " "+secretName) {
			return nil // Secret exists
		}
	}

	return fmt.Errorf("podman secret '%s' does not exist", secretName)
}

// SystemRequirements checks if all required system tools are installed.
// Deprecated: Use NewValidator and Validator.SystemRequirements instead.
func SystemRequirements() error {
	logger := log.NewLogger(false)
	validator := NewValidatorWithDefaults(logger)
	return validator.SystemRequirements()
}
