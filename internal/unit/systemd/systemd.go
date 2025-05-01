// Package systemd handles systemd operations
package systemd

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/config"
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

// getSystemdUnitType returns the appropriate systemd unit type for the given unit type.
// In user mode, container, volume, and network units are registered as regular services.
func getSystemdUnitType(unitType string) string {
	// In user mode, for container, volume and network units, we need to use .service suffix
	// instead of .container, .volume, or .network suffix when referring to the systemd unit
	systemdUnitType := unitType
	if config.GetConfig().UserMode {
		if unitType == "container" || unitType == "volume" || unitType == "network" {
			// In user mode, quadlet units are registered as regular services with adjusted names
			systemdUnitType = "service"

			if config.GetConfig().Verbose {
				log.Printf("In user mode, %s units are registered as .service units", unitType)
			}
		}
	}

	return systemdUnitType
}

// StopSystemdUnit stops a systemd unit.
func StopSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType
	args := []string{"stop", unitName}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user for unit %s", unitName)
		}
	}

	cmd := exec.Command("systemctl", args...) //nolint:gosec // Input is validated
	return cmd.Run()
}

// RestartSystemdUnit restarts a systemd unit.
func RestartSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType
	args := []string{"restart", unitName}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user for unit %s", unitName)
		}
	}

	cmd := exec.Command("systemctl", args...) //nolint:gosec // Input is validated
	return cmd.Run()
}

// ReloadSystemd reloads systemd daemon.
func ReloadSystemd() error {
	args := []string{"daemon-reload"}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user daemon-reload")
		}
	}

	cmd := exec.Command("systemctl", args...)
	return cmd.Run()
}

// GetSystemdUnitStatus gets the status of a systemd unit.
func GetSystemdUnitStatus(name, unitType string) (string, error) {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return "", err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType
	args := []string{"status", "--no-pager", unitName}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user for unit status %s", unitName)
		}
	}

	cmd := exec.Command("systemctl", args...) //nolint:gosec // Input is validated
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// StartSystemdUnit starts a systemd unit.
func StartSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType
	args := []string{"start", unitName}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user for starting unit %s", unitName)
		}
	}

	cmd := exec.Command("systemctl", args...) //nolint:gosec // Input is validated
	return cmd.Run()
}

// ShowSystemdUnit shows the configuration of a systemd unit.
func ShowSystemdUnit(name, unitType string) error {
	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType
	args := []string{"cat", unitName}

	// Use 'systemctl --user' in user mode
	if config.GetConfig().UserMode {
		args = append([]string{"--user"}, args...)
		if config.GetConfig().Verbose {
			log.Printf("Using systemctl --user for showing unit %s", unitName)
		}
	}

	cmd := exec.Command("systemctl", args...) //nolint:gosec // Input is validated
	cmd.Stdout = nil
	return cmd.Run()
}

// ReloadAndStartUnit reloads systemd daemon and starts a unit if it's not already active.
// This is useful after creating or modifying unit files.
func ReloadAndStartUnit(name, unitType string) error {
	// Reload systemd daemon first
	if err := ReloadSystemd(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType

	// Connect to systemd via DBus with context
	ctx := context.Background()
	var conn *dbus.Conn
	var err error
	if config.GetConfig().UserMode {
		// User mode - connect to user's DBus session
		conn, err = dbus.NewUserConnectionContext(ctx)
		if config.GetConfig().Verbose {
			log.Printf("Connecting to user DBus for unit %s", unitName)
		}
	} else {
		// System mode - connect to system DBus
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	// Check if unit is already active
	units, err := conn.ListUnitsByNamesContext(ctx, []string{unitName})
	if err != nil {
		return fmt.Errorf("failed to check unit status: %w", err)
	}

	if len(units) > 0 && units[0].ActiveState == "active" {
		// Unit is already active, no need to start
		if config.GetConfig().Verbose {
			log.Printf("Unit %s is already active, skipping start", unitName)
		}
		return nil
	}

	// Start the unit
	ch := make(chan string)
	_, err = conn.StartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to start unit %s: %w", unitName, err)
	}

	// Wait for job to complete
	jobResult := <-ch
	if jobResult != "done" {
		return fmt.Errorf("failed to start unit %s: job result %s", unitName, jobResult)
	}

	if config.GetConfig().Verbose {
		log.Printf("Successfully started unit %s", unitName)
	}

	return nil
}

// ReloadAndRestartUnit reloads systemd daemon and restarts a unit if it exists.
func ReloadAndRestartUnit(name, unitType string) error {
	// Reload systemd daemon first
	if err := ReloadSystemd(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err := validateUnitNameAndType(name, unitType); err != nil {
		return err
	}

	systemdUnitType := getSystemdUnitType(unitType)
	unitName := name + "." + systemdUnitType

	// Connect to systemd via DBus with context
	ctx := context.Background()
	var conn *dbus.Conn
	var err error
	if config.GetConfig().UserMode {
		// User mode - connect to user's DBus session
		conn, err = dbus.NewUserConnectionContext(ctx)
		if config.GetConfig().Verbose {
			log.Printf("Connecting to user DBus for unit %s", unitName)
		}
	} else {
		// System mode - connect to system DBus
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	// Check if unit exists
	units, err := conn.ListUnitsByNamesContext(ctx, []string{unitName})
	if err != nil {
		return fmt.Errorf("failed to check unit status: %w", err)
	}

	if len(units) == 0 || units[0].LoadState == "not-found" {
		// Unit doesn't exist, try to start it instead
		if config.GetConfig().Verbose {
			log.Printf("Unit %s not found, attempting to start instead of restart", unitName)
		}
		return StartSystemdUnit(name, unitType)
	}

	// Restart the unit
	ch := make(chan string)
	_, err = conn.RestartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to restart unit %s: %w", unitName, err)
	}

	// Wait for job to complete
	jobResult := <-ch
	if jobResult != "done" {
		return fmt.Errorf("failed to restart unit %s: job result %s", unitName, jobResult)
	}

	if config.GetConfig().Verbose {
		log.Printf("Successfully restarted unit %s", unitName)
	}

	return nil
}

// For testing
var (
	ValidateUnitNameAndType = validateUnitNameAndType
	GetSystemdUnitType      = getSystemdUnitType
)