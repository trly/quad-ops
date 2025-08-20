package systemd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/sorting"
	"gopkg.in/ini.v1"

	"github.com/coreos/go-systemd/v22/dbus"
)

// GetStatus returns the current status of the unit.
func (u *BaseUnit) GetStatus() (string, error) {
	conn, err := getSystemdConnection()
	if err != nil {
		return "", fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
	if err != nil {
		return "", fmt.Errorf("error getting unit property: %w", err)
	}
	return prop.Value.Value().(string), nil
}

// Start starts the unit.
func (u *BaseUnit) Start() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	log.GetLogger().Debug("Attempting to start unit", "name", serviceName)

	ch := make(chan string)
	_, err = conn.StartUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error starting unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		// Check if the unit is still in the process of starting up, regardless of result
		activeState, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
		if err == nil && activeState.Value.Value().(string) == "activating" {
			// Get sub-state to understand what kind of activation is happening
			subState, err := conn.GetUnitPropertyContext(ctx, serviceName, "SubState")
			subStateStr := "unknown"
			if err == nil {
				subStateStr = subState.Value.Value().(string)
			}

			log.GetLogger().Debug("Unit is in activating state, waiting for completion",
				"name", serviceName, "subState", subStateStr, "result", result)

			// Wait longer for units that are starting (like downloading images)
			waitTime := config.DefaultProvider().GetConfig().UnitStartTimeout
			if subStateStr == "start" {
				waitTime = config.DefaultProvider().GetConfig().ImagePullTimeout
			}

			time.Sleep(waitTime)

			// Check final state
			finalActiveState, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
			if err == nil {
				finalState := finalActiveState.Value.Value().(string)
				if finalState == "active" {
					log.GetLogger().Info("Unit successfully started after activation delay", "name", serviceName)
					return nil
				}
				log.GetLogger().Debug("Unit not active after waiting", "name", serviceName, "finalState", finalState)
			}
		}

		// Get detailed failure information
		details := getUnitFailureDetails(serviceName)
		return fmt.Errorf("unit start failed: %s\nPossible causes:\n- Missing dependencies\n- Invalid configuration\n- Resource conflicts%s",
			result, details)
	}

	log.GetLogger().Debug("Successfully started unit", "name", serviceName)
	return nil
}

// Stop stops the unit.
func (u *BaseUnit) Stop() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	log.GetLogger().Debug("Attempting to stop unit", "name", serviceName)

	ch := make(chan string)
	_, err = conn.StopUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error stopping unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		// Get detailed failure information
		details := getUnitFailureDetails(serviceName)
		return fmt.Errorf("unit stop failed: %s\nPossible causes:\n- Unit is already stopped\n- Unit has dependent services that need to be stopped first\n- Process is being killed forcefully%s",
			result, details)
	}

	log.GetLogger().Debug("Successfully stopped unit", "name", serviceName)
	return nil
}

// Restart restarts the unit.
func (u *BaseUnit) Restart() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	log.GetLogger().Debug("Attempting to restart unit", "name", serviceName)

	// Check if unit is loaded before attempting restart
	loadState, err := conn.GetUnitPropertyContext(ctx, serviceName, "LoadState")
	if err != nil {
		return fmt.Errorf("error checking unit load state %s: %w", serviceName, err)
	}

	if loadState.Value.Value().(string) != "loaded" {
		return fmt.Errorf("unit %s is not loaded (LoadState: %s), cannot restart", serviceName, loadState.Value.Value().(string))
	}

	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("error restarting unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		// Check if the unit is still in the process of starting up, regardless of result
		activeState, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
		if err == nil && activeState.Value.Value().(string) == "activating" {
			// Get sub-state to understand what kind of activation is happening
			subState, err := conn.GetUnitPropertyContext(ctx, serviceName, "SubState")
			subStateStr := "unknown"
			if err == nil {
				subStateStr = subState.Value.Value().(string)
			}

			log.GetLogger().Debug("Unit is in activating state, waiting for completion",
				"name", serviceName, "subState", subStateStr, "result", result)

			// Wait longer for units that are starting (like downloading images)
			waitTime := 5 * time.Second
			if subStateStr == "start" {
				waitTime = 10 * time.Second // More time for container image pulls, etc.
			}

			time.Sleep(waitTime)

			// Check final state
			finalActiveState, err := conn.GetUnitPropertyContext(ctx, serviceName, "ActiveState")
			if err == nil {
				finalState := finalActiveState.Value.Value().(string)
				if finalState == "active" {
					log.GetLogger().Info("Unit successfully restarted after activation delay", "name", serviceName)
					return nil
				}
				log.GetLogger().Debug("Unit not active after waiting", "name", serviceName, "finalState", finalState)
			}
		}

		// Get detailed failure information
		details := getUnitFailureDetails(serviceName)
		return fmt.Errorf("unit restart failed: %s\nReason: dependency issues or unit configuration errors%s",
			result, details)
	}

	log.GetLogger().Debug("Successfully restarted unit", "name", serviceName)
	return nil
}

// Show displays the unit configuration and status.
func (u *BaseUnit) Show() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitPropertiesContext(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("error getting unit properties: %w", err)
	}

	fmt.Printf("\n=== %s ===\n\n", serviceName)

	fmt.Println("Status:")
	fmt.Printf("  %-20s: %v\n", "State", prop["ActiveState"])
	fmt.Printf("  %-20s: %v\n", "Sub-State", prop["SubState"])
	fmt.Printf("  %-20s: %v\n", "Load State", prop["LoadState"])

	fmt.Println("\nUnit Information:")
	fmt.Printf("  %-20s: %v\n", "Description", prop["Description"])
	fmt.Printf("  %-20s: %v\n", "Path", prop["FragmentPath"])

	// Read and display the actual quadlet configuration
	if fragmentPath, ok := prop["FragmentPath"].(string); ok {
		content, err := os.ReadFile(fragmentPath) //nolint:gosec // Safe as path comes from systemd D-Bus interface, not user input
		if err == nil {
			unitConfig, _ := ini.Load(content)
			quadletSectionName := fmt.Sprintf("X-%s", caser.String(u.Type))
			if section, err := unitConfig.GetSection(quadletSectionName); err == nil {
				fmt.Printf("\n%s Configuration:\n", caser.String(u.Type))
				for _, key := range section.Keys() {
					fmt.Printf("  %-20s: %s\n", key.Name(), key.Value())
				}
			}
			if section, err := unitConfig.GetSection("Service"); err == nil {
				fmt.Printf("\n%s Configuration:\n", caser.String("service"))
				for _, key := range section.Keys() {
					if key.Name() == "ExecStart" {
						fmt.Printf("  %-20s: %s\n", key.Name(), key.Value())
					}
				}
			}
		}
	}

	fmt.Println()
	return nil
}

// ResetFailed resets the failed state of the unit.
func (u *BaseUnit) ResetFailed() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := u.GetServiceName()
	log.GetLogger().Debug("Resetting failed unit", "name", serviceName)
	err = conn.ResetFailedUnitContext(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("error resetting failed unit %s: %w", serviceName, err)
	}

	return nil
}

// ReloadSystemd reloads the systemd configuration.
func ReloadSystemd() error {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	log.GetLogger().Debug("Reloading systemd")
	err = conn.ReloadContext(ctx)
	if err != nil {
		return fmt.Errorf("error reloading systemd: %w", err)
	}

	return nil
}

// Utility functions

// GetSystemdConnection returns a connection to systemd D-Bus.
func GetSystemdConnection() (*dbus.Conn, error) {
	return getSystemdConnection()
}

// GetContext returns the systemd operation context.
func GetContext() context.Context {
	return ctx
}

// GetUnitFailureDetails retrieves additional details about a unit failure using dbus.
func GetUnitFailureDetails(unitName string) string {
	return getUnitFailureDetails(unitName)
}

// getUnitFailureDetails retrieves additional details about a unit failure using dbus.
func getUnitFailureDetails(unitName string) string {
	conn, err := getSystemdConnection()
	if err != nil {
		return fmt.Sprintf("Could not connect to systemd: %v", err)
	}
	defer conn.Close()

	// Get unit properties via dbus instead of exec.Command
	prop, err := conn.GetUnitPropertiesContext(ctx, unitName)
	if err != nil {
		return fmt.Sprintf("Could not retrieve unit properties: %v", err)
	}

	// Build status information from dbus properties
	statusInfo := fmt.Sprintf("Unit: %s\n", unitName)
	statusInfo += fmt.Sprintf("  Load State: %v\n", prop["LoadState"])
	statusInfo += fmt.Sprintf("  Active State: %v\n", prop["ActiveState"])
	statusInfo += fmt.Sprintf("  Sub State: %v\n", prop["SubState"])

	if result, ok := prop["Result"]; ok {
		statusInfo += fmt.Sprintf("  Result: %v\n", result)
	}

	if mainPID, ok := prop["MainPID"]; ok && mainPID != uint32(0) {
		statusInfo += fmt.Sprintf("  Main PID: %v\n", mainPID)
	}

	if execMainStatus, ok := prop["ExecMainStatus"]; ok {
		statusInfo += fmt.Sprintf("  Exit Status: %v\n", execMainStatus)
	}

	// For logs, we still need journalctl as systemd dbus doesn't provide log retrieval
	// This is the only remaining exec.Command, but it's necessary as dbus doesn't expose logs

	// Validate unitName to prevent command injection
	if err := sorting.ValidateUnitName(unitName); err != nil {
		return fmt.Sprintf("\nUnit Status (via dbus):\n%s\nRecent logs: (unavailable - invalid unit name)", statusInfo)
	}

	cmd := exec.Command("journalctl", "--user-unit", unitName, "-n", "3", "--no-pager", "--output=short-precise")
	if !config.DefaultProvider().GetConfig().UserMode {
		cmd = exec.Command("journalctl", "--unit", unitName, "-n", "3", "--no-pager", "--output=short-precise")
	}
	output, err := cmd.CombinedOutput()
	logInfo := "Recent logs: (unavailable)"
	if err == nil && len(output) > 0 {
		logInfo = fmt.Sprintf("Recent logs:\n%s", string(output))
	}

	return fmt.Sprintf("\nUnit Status (via dbus):\n%s\n%s", statusInfo, logInfo)
}

func getSystemdConnection() (*dbus.Conn, error) {
	if config.DefaultProvider().GetConfig().UserMode {
		log.GetLogger().Debug("Establishing user connection to systemd")
	} else {
		log.GetLogger().Debug("Establishing system connection to systemd")
	}

	if config.DefaultProvider().GetConfig().UserMode {
		return dbus.NewUserConnectionContext(ctx)
	}
	return dbus.NewSystemConnectionContext(ctx)
}
