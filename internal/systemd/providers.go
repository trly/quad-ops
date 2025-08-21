package systemd

import (
	"context"
	"fmt"

	"os/exec"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/sorting"
)

// DefaultContextProvider implements ContextProvider interface.
type DefaultContextProvider struct {
	ctx context.Context
}

// NewDefaultContextProvider creates a new default context provider.
func NewDefaultContextProvider() *DefaultContextProvider {
	return &DefaultContextProvider{
		ctx: context.Background(),
	}
}

// GetContext returns a context for systemd operations.
func (p *DefaultContextProvider) GetContext() context.Context {
	return p.ctx
}

// DefaultTextCaser implements TextCaser interface.
type DefaultTextCaser struct {
	caser cases.Caser
}

// NewDefaultTextCaser creates a new default text caser.
func NewDefaultTextCaser() *DefaultTextCaser {
	return &DefaultTextCaser{
		caser: cases.Title(language.English),
	}
}

// Title converts text to title case.
func (c *DefaultTextCaser) Title(text string) string {
	return c.caser.String(text)
}

// DefaultUnitManager implements UnitManager interface.
type DefaultUnitManager struct {
	connectionFactory ConnectionFactory
	contextProvider   ContextProvider
	textCaser         TextCaser
}

// NewDefaultUnitManager creates a new default unit manager.
func NewDefaultUnitManager(connectionFactory ConnectionFactory, contextProvider ContextProvider, textCaser TextCaser) *DefaultUnitManager {
	return &DefaultUnitManager{
		connectionFactory: connectionFactory,
		contextProvider:   contextProvider,
		textCaser:         textCaser,
	}
}

// GetUnit creates a Unit interface for the given name and type.
func (m *DefaultUnitManager) GetUnit(name, unitType string) Unit {
	return &ManagedUnit{
		BaseUnit:          NewBaseUnit(name, unitType),
		connectionFactory: m.connectionFactory,
		contextProvider:   m.contextProvider,
		textCaser:         m.textCaser,
	}
}

// GetStatus returns the current status of a unit.
func (m *DefaultUnitManager) GetStatus(unitName, unitType string) (string, error) {
	unit := m.GetUnit(unitName, unitType)
	return unit.GetStatus()
}

// Start starts a unit.
func (m *DefaultUnitManager) Start(unitName, unitType string) error {
	unit := m.GetUnit(unitName, unitType)
	return unit.Start()
}

// Stop stops a unit.
func (m *DefaultUnitManager) Stop(unitName, unitType string) error {
	unit := m.GetUnit(unitName, unitType)
	return unit.Stop()
}

// Restart restarts a unit.
func (m *DefaultUnitManager) Restart(unitName, unitType string) error {
	unit := m.GetUnit(unitName, unitType)
	return unit.Restart()
}

// Show displays unit configuration and status.
func (m *DefaultUnitManager) Show(unitName, unitType string) error {
	unit := m.GetUnit(unitName, unitType)
	return unit.Show()
}

// ResetFailed resets the failed state of a unit.
func (m *DefaultUnitManager) ResetFailed(unitName, unitType string) error {
	unit := m.GetUnit(unitName, unitType)
	return unit.ResetFailed()
}

// ReloadSystemd reloads systemd configuration.
func (m *DefaultUnitManager) ReloadSystemd() error {
	conn, err := m.connectionFactory.NewConnection(m.contextProvider.GetContext(), config.DefaultProvider().GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	log.GetLogger().Debug("Reloading systemd")
	return conn.Reload(m.contextProvider.GetContext())
}

// GetUnitFailureDetails gets detailed failure information for a unit.
func (m *DefaultUnitManager) GetUnitFailureDetails(unitName string) string {
	return m.getUnitFailureDetails(unitName)
}

// getUnitFailureDetails retrieves additional details about a unit failure.
func (m *DefaultUnitManager) getUnitFailureDetails(unitName string) string {
	conn, err := m.connectionFactory.NewConnection(m.contextProvider.GetContext(), config.DefaultProvider().GetConfig().UserMode)
	if err != nil {
		return fmt.Sprintf("Could not connect to systemd: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Get unit properties
	prop, err := conn.GetUnitProperties(m.contextProvider.GetContext(), unitName)
	if err != nil {
		return fmt.Sprintf("Could not retrieve unit properties: %v", err)
	}

	// Build status information from properties
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

// DefaultOrchestrator implements Orchestrator interface.
type DefaultOrchestrator struct {
	unitManager UnitManager
}

// NewDefaultOrchestrator creates a new default orchestrator.
func NewDefaultOrchestrator(unitManager UnitManager) *DefaultOrchestrator {
	return &DefaultOrchestrator{
		unitManager: unitManager,
	}
}

// StartUnitDependencyAware starts or restarts a unit with dependency awareness.
func (o *DefaultOrchestrator) StartUnitDependencyAware(unitName, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error {
	log.GetLogger().Debug("Starting/restarting unit with dependency awareness", "unit", unitName, "type", unitType)

	// Handle different unit types appropriately
	switch unitType {
	case "build", "image", "network", "volume":
		// For one-shot services, use Start() instead of Restart()
		log.GetLogger().Debug("Starting one-shot service", "unit", unitName, "type", unitType)
		return o.unitManager.Start(unitName, unitType)
	}

	// Only handle containers for dependency logic
	if unitType != "container" {
		// For other non-container units, just use the normal restart method
		log.GetLogger().Debug("Direct restart for non-container unit", "unit", unitName, "type", unitType)
		return o.unitManager.Restart(unitName, unitType)
	}

	// For containers, use the dependency-aware restart
	// Parse the unitName to get the service name
	parts := splitUnitName(unitName)
	if len(parts) != 2 {
		// Invalid unit name format, fall back to regular restart
		log.GetLogger().Warn("Invalid unit name format, using direct restart",
			"unit", unitName,
			"expected", "project-service")
		return o.unitManager.Restart(unitName, unitType)
	}

	projectName := parts[0]
	serviceName := parts[1]

	// Check if this service is in the dependency graph
	dependencies, err := dependencyGraph.GetDependencies(serviceName)
	if err != nil {
		// Service not in dependency graph, fall back to regular restart
		log.GetLogger().Debug("Service not found in dependency graph, using direct restart",
			"service", serviceName,
			"project", projectName,
			"error", err)
		return o.unitManager.Restart(unitName, unitType)
	}

	// Always restart the changed service directly - systemd will handle dependency propagation
	log.GetLogger().Debug("Restarting changed service directly - systemd will handle dependency propagation",
		"service", serviceName,
		"project", projectName,
		"dependencies", dependencies)

	return o.unitManager.Restart(unitName, unitType)
}

// RestartChangedUnits restarts all changed units in dependency-aware order.
func (o *DefaultOrchestrator) RestartChangedUnits(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	log.GetLogger().Info("Restarting changed units with dependency awareness", "count", len(changedUnits))

	err := o.unitManager.ReloadSystemd()
	if err != nil {
		log.GetLogger().Error("Failed to reload systemd units", "error", err)
		return fmt.Errorf("failed to reload systemd configuration: %w", err)
	}

	// Wait for systemd to process the changes
	time.Sleep(2 * time.Second)

	// Track units with restart failures
	restartFailures := make(map[string]error)

	// First start one-shot services (network, volume, build, image)
	log.GetLogger().Debug("Starting one-shot services first")
	for _, unit := range changedUnits {
		switch unit.Type {
		case "network", "volume", "build", "image":
			unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)

			// For volumes, warn about potential data safety considerations
			if unit.Type == "volume" {
				log.GetLogger().Debug("Starting volume unit - existing data should be preserved",
					"name", unit.Name,
					"note", "Podman typically preserves existing volume data when configuration changes")
			}

			// Use Start() for one-shot services since they don't run continuously
			if err := unit.Unit.Start(); err != nil {
				log.GetLogger().Error("Failed to start one-shot service",
					"type", unit.Type,
					"name", unit.Name,
					"error", err)
				restartFailures[unitKey] = err
			} else {
				log.GetLogger().Debug("Successfully started one-shot service", "name", unit.Name, "type", unit.Type)
			}
		}
	}

	// Wait for one-shot services to complete
	time.Sleep(1 * time.Second)

	// Phase 1: Initiate all container restarts asynchronously
	log.GetLogger().Debug("Initiating async restarts for container units")
	containerUnits := make([]UnitChange, 0, len(changedUnits))
	restarted := make(map[string]bool)

	for _, unit := range changedUnits {
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
		// Skip one-shot services (already handled) and non-container units
		if unit.Type == "network" || unit.Type == "volume" || unit.Type == "build" || unit.Type == "image" {
			continue
		}

		if unit.Type != "container" {
			// For non-container units, restart synchronously since they're usually quick
			log.GetLogger().Debug("Using direct restart for non-container unit", "name", unit.Name, "type", unit.Type)
			err := unit.Unit.Restart()
			if err != nil {
				log.GetLogger().Error("Failed to restart non-container unit",
					"name", unit.Name, "type", unit.Type, "error", err)
				restartFailures[unitKey] = err
			}
			continue
		}

		// For container units, initiate restart without waiting
		containerUnits = append(containerUnits, unit)

		// Check if we need dependency-aware restart
		parts := splitUnitName(unit.Name)
		if len(parts) == 2 {
			projectName := parts[0]
			serviceName := parts[1]

			if dependencyGraph, ok := projectDependencyGraphs[projectName]; ok {
				if isServiceAlreadyRestarted(serviceName, dependencyGraph, projectName, restarted) {
					log.GetLogger().Debug("Skipping restart as unit or its dependent services were already restarted",
						"name", unit.Name, "project", projectName, "service", serviceName)
					continue
				}
			}
		}

		// Initiate the restart asynchronously using systemd's async restart
		log.GetLogger().Debug("Initiating async restart", "name", unit.Name)
		err := o.initiateAsyncRestart(unit.Unit)
		if err != nil {
			log.GetLogger().Error("Failed to initiate restart",
				"name", unit.Name, "error", err)
			restartFailures[unitKey] = err
		}
	}

	// Phase 2: Wait for all restarts to complete and check status
	if len(containerUnits) > 0 {
		log.GetLogger().Info("Waiting for container units to complete restart", "count", len(containerUnits))
		time.Sleep(5 * time.Second) // Initial wait for restarts to begin

		// Check each container unit's final status
		for _, unit := range containerUnits {
			unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)

			// Skip if we already marked this as failed during initiation
			if _, failed := restartFailures[unitKey]; failed {
				continue
			}

			// Check final status with activating state handling
			err := o.checkUnitFinalStatus(unit.Unit)
			if err != nil {
				log.GetLogger().Error("Unit failed to reach active state",
					"name", unit.Name, "error", err)
				restartFailures[unitKey] = err
			} else {
				log.GetLogger().Debug("Unit successfully restarted", "name", unit.Name)
			}
		}
	}

	// Summarize restart failures if any occurred
	if len(restartFailures) > 0 {
		// Log all failures individually
		for unit, unitErr := range restartFailures {
			log.GetLogger().Error("Unit restart failure", "unit", unit, "error", unitErr)
		}
		log.GetLogger().Error("Some units failed to restart", "count", len(restartFailures))

		// Get the first failing unit for the error message
		firstUnit := ""
		for unit := range restartFailures {
			firstUnit = unit
			break
		}

		return fmt.Errorf("failed to restart %d units. Review logs for details. First error for %s: %v",
			len(restartFailures), firstUnit, restartFailures[firstUnit])
	}

	log.GetLogger().Info("Successfully restarted all changed units", "count", len(changedUnits))
	return nil
}

// initiateAsyncRestart starts a unit restart without waiting for completion.
func (o *DefaultOrchestrator) initiateAsyncRestart(unit Unit) error {
	conn, err := GetConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := unit.GetServiceName()
	log.GetLogger().Debug("Initiating async restart", "name", serviceName)

	ctx := context.Background()

	// Check if unit is loaded before attempting restart
	loadState, err := conn.GetUnitProperty(ctx, serviceName, "LoadState")
	if err != nil {
		return fmt.Errorf("error checking unit load state %s: %w", serviceName, err)
	}

	if loadState.Value.Value().(string) != "loaded" {
		return fmt.Errorf("unit %s is not loaded (LoadState: %s), cannot restart", serviceName, loadState.Value.Value().(string))
	}

	// Initiate restart without waiting for completion
	ch, err := conn.RestartUnit(ctx, serviceName, "replace")
	if err != nil {
		return fmt.Errorf("error initiating restart for unit %s: %w", serviceName, err)
	}

	// Read the immediate result but don't block on completion
	go func() {
		result := <-ch
		log.GetLogger().Debug("Restart initiation result", "name", serviceName, "result", result)
	}()

	return nil
}

// checkUnitFinalStatus checks if a unit has reached active state, with handling for activating states.
func (o *DefaultOrchestrator) checkUnitFinalStatus(unit Unit) error {
	conn, err := GetConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := unit.GetServiceName()
	ctx := context.Background()

	// Check current state
	activeState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
	if err != nil {
		return fmt.Errorf("error getting unit state %s: %w", serviceName, err)
	}

	currentState := activeState.Value.Value().(string)
	if currentState == "active" {
		return nil // Already active, success
	}

	// If it's activating, wait and check again
	if currentState == "activating" {
		subState, err := conn.GetUnitProperty(ctx, serviceName, "SubState")
		subStateStr := "unknown"
		if err == nil {
			subStateStr = subState.Value.Value().(string)
		}

		log.GetLogger().Debug("Unit still activating, waiting for completion",
			"name", serviceName, "subState", subStateStr)

		// Wait based on sub-state
		waitTime := config.DefaultProvider().GetConfig().UnitStartTimeout
		if subStateStr == "start" {
			waitTime = config.DefaultProvider().GetConfig().ImagePullTimeout
		}

		time.Sleep(waitTime)

		// Check final state
		finalActiveState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
		if err == nil {
			finalState := finalActiveState.Value.Value().(string)
			if finalState == "active" {
				log.GetLogger().Info("Unit successfully reached active state", "name", serviceName)
				return nil
			}
			currentState = finalState
		}
	}

	// If we get here, the unit is not active
	details := o.unitManager.GetUnitFailureDetails(serviceName)
	return fmt.Errorf("unit failed to reach active state: %s%s", currentState, details)
}
