// Package systemd handles systemd operations
package systemd

import (
	"os/exec"
	"strings"
)

// StopSystemdUnit stops a systemd unit.
func StopSystemdUnit(name, unitType string) error {
	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "stop", unitName)
	return cmd.Run()
}

// RestartSystemdUnit restarts a systemd unit.
func RestartSystemdUnit(name, unitType string) error {
	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "restart", unitName)
	return cmd.Run()
}

// ReloadSystemd reloads systemd daemon.
func ReloadSystemd() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

// GetSystemdUnitStatus gets the status of a systemd unit.
func GetSystemdUnitStatus(name, unitType string) (string, error) {
	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "status", "--no-pager", unitName)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// StartSystemdUnit starts a systemd unit.
func StartSystemdUnit(name, unitType string) error {
	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "start", unitName)
	return cmd.Run()
}

// ShowSystemdUnit shows the configuration of a systemd unit.
func ShowSystemdUnit(name, unitType string) error {
	unitName := name + "." + unitType
	cmd := exec.Command("systemctl", "cat", unitName)
	cmd.Stdout = nil 
	return cmd.Run()
}