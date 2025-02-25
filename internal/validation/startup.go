package validation

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func VerifySystemRequirements(verbose bool) error {

	if verbose {
		log.Print("balidate systemd is available")
	}
	systemdVersion, err := exec.Command("systemctl", "--version").Output()
	if err != nil {
		return fmt.Errorf("systemd not found: %w", err)
	}

	if !strings.Contains(string(systemdVersion), "systemd") {
		return fmt.Errorf("systemd not properly installed")
	}

	if verbose {
		log.Print("validate podman is available")
	}
	_, err = exec.Command("podman", "--version").Output()
	if err != nil {
		return fmt.Errorf("podman not found: %w", err)
	}

	if verbose {
		log.Print("balidate podman-system-generator is available")
	}
	generatorPath := "/usr/lib/systemd/system-generators/podman-system-generator"
	_, err = exec.Command("test", "-f", generatorPath).Output()
	if err != nil {
		return fmt.Errorf("podman systemd generator not found at %s", generatorPath)
	}

	return nil
}
