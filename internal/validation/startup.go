package validation

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/trly/quad-ops/internal/config"
)

func VerifySystemRequirements(cfg config.Config) error {

	if cfg.Verbose {
		log.Print("validate systemd is available")
	}

	systemdVersion, err := exec.Command("systemctl", "--version").Output()
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	if cfg.Verbose {
		log.Print("validate podman is available")
	}

	_, err = exec.Command("podman", "--version").Output()
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	if cfg.Verbose {
		log.Print("validate podman-system-generator is available")
	}

	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = exec.Command("test", "-f", generatorPath).Output()
	if err != nil {
		return fmt.Errorf("podman systemd generator not found at %s", generatorPath)
	}

	return nil
}
