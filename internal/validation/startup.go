// validation/validation.go
package validation

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/trly/quad-ops/internal/config"
)

// CommandRunner defines an interface for executing commands
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner implements CommandRunner using os/exec
type RealCommandRunner struct{}

// Run executes a command and returns its output
func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// default runner for use in production code
var defaultRunner CommandRunner = &RealCommandRunner{}

// SetCommandRunner allows tests to inject a mock runner
func SetCommandRunner(runner CommandRunner) {
	defaultRunner = runner
}

// ResetCommandRunner restores the default runner
func ResetCommandRunner() {
	defaultRunner = &RealCommandRunner{}
}

// VerifySystemRequirements checks if all required system tools are installed
func VerifySystemRequirements() error {
	if config.GetConfig().Verbose {
		log.Print("validate systemd is available")
	}

	systemdVersion, err := defaultRunner.Run("systemctl", "--version")
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	if config.GetConfig().Verbose {
		log.Print("validate podman is available")
	}

	_, err = defaultRunner.Run("podman", "--version")
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	if config.GetConfig().Verbose {
		log.Print("validate podman-system-generator is available")
	}

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = defaultRunner.Run("test", "-f", generatorPath)
	if err != nil {
		return fmt.Errorf("podman systemd generator not found at %s", generatorPath)
	}

	return nil
}
