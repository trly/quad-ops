package systemd

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/ini.v1"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

// ManagedUnit wraps BaseUnit with injected dependencies for testing.
type ManagedUnit struct {
	*BaseUnit
	connectionFactory ConnectionFactory
	contextProvider   ContextProvider
	textCaser         TextCaser
	configProvider    config.Provider
	logger            log.Logger
}

// NewManagedUnit creates a new managed unit with injected dependencies.
func NewManagedUnit(name, unitType string, connectionFactory ConnectionFactory, contextProvider ContextProvider, textCaser TextCaser, configProvider config.Provider, logger log.Logger) *ManagedUnit {
	return &ManagedUnit{
		BaseUnit:          NewBaseUnit(name, unitType),
		connectionFactory: connectionFactory,
		contextProvider:   contextProvider,
		textCaser:         textCaser,
		configProvider:    configProvider,
		logger:            logger,
	}
}

// GetStatus returns the current status of the unit.
func (u *ManagedUnit) GetStatus() (string, error) {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return "", err // Connection factory already wraps with proper error type
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitProperty(u.contextProvider.GetContext(), serviceName, "ActiveState")
	if err != nil {
		return "", NewError("GetStatus", u.Name, u.Type, err)
	}
	return prop.Value.Value().(string), nil
}

// Start starts the unit.
func (u *ManagedUnit) Start() error {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	u.logger.Debug("Attempting to start unit", "name", serviceName)

	ch, err := conn.StartUnit(u.contextProvider.GetContext(), serviceName, "replace")
	if err != nil {
		return fmt.Errorf("error starting unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		// Check if the unit is still in the process of starting up, regardless of result
		activeState, err := conn.GetUnitProperty(u.contextProvider.GetContext(), serviceName, "ActiveState")
		if err == nil && activeState.Value.Value().(string) == "activating" {
			// Get sub-state to understand what kind of activation is happening
			subState, err := conn.GetUnitProperty(u.contextProvider.GetContext(), serviceName, "SubState")
			subStateStr := "unknown"
			if err == nil {
				subStateStr = subState.Value.Value().(string)
			}

			u.logger.Debug("Unit is in activating state, waiting for completion",
				"name", serviceName, "subState", subStateStr, "result", result)

			// Wait longer for units that are starting (like downloading images)
			waitTime := u.configProvider.GetConfig().UnitStartTimeout
			if subStateStr == "start" {
				waitTime = u.configProvider.GetConfig().ImagePullTimeout
			}

			time.Sleep(waitTime)

			// Check final state
			finalActiveState, err := conn.GetUnitProperty(u.contextProvider.GetContext(), serviceName, "ActiveState")
			if err == nil {
				finalState := finalActiveState.Value.Value().(string)
				if finalState == "active" {
					u.logger.Info("Unit successfully started after activation delay", "name", serviceName)
					return nil
				}
				u.logger.Debug("Unit not active after waiting", "name", serviceName, "finalState", finalState)
			}
		}

		// Get detailed failure information - for now use a simple approach
		return fmt.Errorf("unit start failed: %s\nPossible causes:\n- Missing dependencies\n- Invalid configuration\n- Resource conflicts",
			result)
	}

	u.logger.Debug("Successfully started unit", "name", serviceName)
	return nil
}

// Stop stops the unit.
func (u *ManagedUnit) Stop() error {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	u.logger.Debug("Attempting to stop unit", "name", serviceName)

	ch, err := conn.StopUnit(u.contextProvider.GetContext(), serviceName, "replace")
	if err != nil {
		return fmt.Errorf("error stopping unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		return fmt.Errorf("unit stop failed: %s\nPossible causes:\n- Unit is already stopped\n- Unit has dependent services that need to be stopped first\n- Process is being killed forcefully",
			result)
	}

	u.logger.Debug("Successfully stopped unit", "name", serviceName)
	return nil
}

// Restart restarts the unit.
func (u *ManagedUnit) Restart() error {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	u.logger.Debug("Attempting to restart unit", "name", serviceName)

	ctx := u.contextProvider.GetContext()

	// Check if unit is loaded before attempting restart
	loadState, err := conn.GetUnitProperty(ctx, serviceName, "LoadState")
	if err != nil {
		return fmt.Errorf("error checking unit load state %s: %w", serviceName, err)
	}

	if loadState.Value.Value().(string) != "loaded" {
		return fmt.Errorf("unit %s is not loaded (LoadState: %s), cannot restart", serviceName, loadState.Value.Value().(string))
	}

	ch, err := conn.RestartUnit(ctx, serviceName, "replace")
	if err != nil {
		return fmt.Errorf("error restarting unit %s: %w", serviceName, err)
	}

	result := <-ch
	if result != "done" {
		// Check if the unit is still in the process of starting up, regardless of result
		activeState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
		if err == nil && activeState.Value.Value().(string) == "activating" {
			// Get sub-state to understand what kind of activation is happening
			subState, err := conn.GetUnitProperty(ctx, serviceName, "SubState")
			subStateStr := "unknown"
			if err == nil {
				subStateStr = subState.Value.Value().(string)
			}

			u.logger.Debug("Unit is in activating state, waiting for completion",
				"name", serviceName, "subState", subStateStr, "result", result)

			// Wait longer for units that are starting (like downloading images)
			waitTime := 5 * time.Second
			if subStateStr == "start" {
				waitTime = 10 * time.Second // More time for container image pulls, etc.
			}

			time.Sleep(waitTime)

			// Check final state
			finalActiveState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
			if err == nil {
				finalState := finalActiveState.Value.Value().(string)
				if finalState == "active" {
					u.logger.Info("Unit successfully restarted after activation delay", "name", serviceName)
					return nil
				}
				u.logger.Debug("Unit not active after waiting", "name", serviceName, "finalState", finalState)
			}
		}

		return fmt.Errorf("unit restart failed: %s\nReason: dependency issues or unit configuration errors",
			result)
	}

	u.logger.Debug("Successfully restarted unit", "name", serviceName)
	return nil
}

// Show displays the unit configuration and status.
func (u *ManagedUnit) Show() error {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	prop, err := conn.GetUnitProperties(u.contextProvider.GetContext(), serviceName)
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
			quadletSectionName := fmt.Sprintf("X-%s", u.textCaser.Title(u.Type))
			if section, err := unitConfig.GetSection(quadletSectionName); err == nil {
				fmt.Printf("\n%s Configuration:\n", u.textCaser.Title(u.Type))
				for _, key := range section.Keys() {
					fmt.Printf("  %-20s: %s\n", key.Name(), key.Value())
				}
			}
			if section, err := unitConfig.GetSection("Service"); err == nil {
				fmt.Printf("\n%s Configuration:\n", u.textCaser.Title("service"))
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
func (u *ManagedUnit) ResetFailed() error {
	conn, err := u.connectionFactory.NewConnection(u.contextProvider.GetContext(), u.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := u.GetServiceName()
	u.logger.Debug("Resetting failed unit", "name", serviceName)
	err = conn.ResetFailedUnit(u.contextProvider.GetContext(), serviceName)
	if err != nil {
		return fmt.Errorf("error resetting failed unit %s: %w", serviceName, err)
	}

	return nil
}
