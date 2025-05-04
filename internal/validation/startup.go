// Package validation provides application validation functionality
package validation

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/trly/quad-ops/internal/logger"
)

// CommandRunner defines an interface for executing commands.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

// Run executes a command and returns its output.
func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
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

// VerifySystemRequirements checks if all required system tools are installed.
func VerifySystemRequirements() error {
	logger.GetLogger().Debug("Validating systemd availability")

	systemdVersion, err := defaultRunner.Run("systemctl", "--version")
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	logger.GetLogger().Debug("Validating podman availability")

	_, err = defaultRunner.Run("podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	logger.GetLogger().Debug("Validating podman-system-generator availability")

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = defaultRunner.Run("test", "-f", generatorPath)
	if err != nil {
		return fmt.Errorf("podman systemd generator not found at %s", generatorPath)
	}

	return nil
}
