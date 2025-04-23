// Package systemd handles systemd operations
package systemd

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// validUnitNamePattern defines allowed characters in systemd unit names.
	validUnitNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9@_.-]*$`)

	// validUnitTypePattern defines allowed characters in systemd unit types.
	validUnitTypePattern = regexp.MustCompile(`^[a-z]+$`)
)

// validateUnitNameAndType validates that systemd unit name and type contain only allowed characters.
func validateUnitNameAndType(name, unitType string) error {
	if !validUnitNamePattern.MatchString(name) {
		return fmt.Errorf("invalid unit name %q, must contain only alphanumeric characters", name)
	}

	if !validUnitTypePattern.MatchString(unitType) {
		return fmt.Errorf("invalid unit type %q, must contain only lowercase letters", unitType)
	}

	return nil
}

// StopSystemdUnit stops a systemd unit.
func StopSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "stop", unitName) //nolint:gosec // Input is validated
	return cmd.Run()
}

// RestartSystemdUnit restarts a systemd unit.
func RestartSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "restart", unitName) //nolint:gosec // Input is validated
	return cmd.Run()
}

// ReloadSystemd reloads systemd daemon.
func ReloadSystemd() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

// GetSystemdUnitStatus gets the status of a systemd unit.
func GetSystemdUnitStatus(name, unitType string) (string, error) {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return "", err
	}

	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "status", "--no-pager", unitName) //nolint:gosec // Input is validated
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// StartSystemdUnit starts a systemd unit.
func StartSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "start", unitName) //nolint:gosec // Input is validated
	return cmd.Run()
}

// ShowSystemdUnit shows the configuration of a systemd unit.
func ShowSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "cat", unitName) //nolint:gosec // Input is validated
	cmd.Stdout = nil
	return cmd.Run()
}
