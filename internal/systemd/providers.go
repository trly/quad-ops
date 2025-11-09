package systemd

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/execx"
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
	configProvider    config.Provider
	logger            log.Logger
	runner            execx.Runner
}

// NewDefaultUnitManager creates a new default unit manager.
func NewDefaultUnitManager(connectionFactory ConnectionFactory, contextProvider ContextProvider, textCaser TextCaser, configProvider config.Provider, logger log.Logger, runner execx.Runner) *DefaultUnitManager {
	return &DefaultUnitManager{
		connectionFactory: connectionFactory,
		contextProvider:   contextProvider,
		textCaser:         textCaser,
		configProvider:    configProvider,
		logger:            logger,
		runner:            runner,
	}
}

// GetUnit creates a Unit interface for the given name and type.
func (m *DefaultUnitManager) GetUnit(name, unitType string) Unit {
	return &ManagedUnit{
		BaseUnit:          NewBaseUnit(name, unitType),
		connectionFactory: m.connectionFactory,
		contextProvider:   m.contextProvider,
		textCaser:         m.textCaser,
		configProvider:    m.configProvider,
		logger:            m.logger,
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
	conn, err := m.connectionFactory.NewConnection(m.contextProvider.GetContext(), m.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	m.logger.Debug("Reloading systemd")
	return conn.Reload(m.contextProvider.GetContext())
}

// GetUnitFailureDetails gets detailed failure information for a unit.
func (m *DefaultUnitManager) GetUnitFailureDetails(unitName string) string {
	return m.getUnitFailureDetails(unitName)
}

// getUnitFailureDetails retrieves additional details about a unit failure.
func (m *DefaultUnitManager) getUnitFailureDetails(unitName string) string {
	conn, err := m.connectionFactory.NewConnection(m.contextProvider.GetContext(), m.configProvider.GetConfig().UserMode)
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

	ctx := context.Background()
	var output []byte
	if m.configProvider.GetConfig().UserMode {
		output, err = m.runner.CombinedOutput(ctx, "journalctl", "--user-unit", unitName, "-n", "3", "--no-pager", "--output=short-precise")
	} else {
		output, err = m.runner.CombinedOutput(ctx, "journalctl", "--unit", unitName, "-n", "3", "--no-pager", "--output=short-precise")
	}

	logInfo := "Recent logs: (unavailable)"
	if err == nil && len(output) > 0 {
		logInfo = fmt.Sprintf("Recent logs:\n%s", string(output))
	}

	return fmt.Sprintf("\nUnit Status (via dbus):\n%s\n%s", statusInfo, logInfo)
}

// DefaultOrchestrator implements Orchestrator interface.
type DefaultOrchestrator struct {
	unitManager       UnitManager
	connectionFactory ConnectionFactory
	configProvider    config.Provider
	logger            log.Logger
}

// NewDefaultOrchestrator creates a new default orchestrator.
func NewDefaultOrchestrator(unitManager UnitManager, connectionFactory ConnectionFactory, configProvider config.Provider, logger log.Logger) *DefaultOrchestrator {
	return &DefaultOrchestrator{
		unitManager:       unitManager,
		connectionFactory: connectionFactory,
		configProvider:    configProvider,
		logger:            logger,
	}
}

// StartUnitDependencyAware starts or restarts a unit with dependency awareness.
func (o *DefaultOrchestrator) StartUnitDependencyAware(unitName, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error {
	o.logger.Debug("Starting/restarting unit with dependency awareness", "unit", unitName, "type", unitType)

	// Handle different unit types appropriately
	switch unitType {
	case "build", "image", "network", "volume":
		// For one-shot services, use Start() instead of Restart()
		o.logger.Debug("Starting one-shot service", "unit", unitName, "type", unitType)
		return o.unitManager.Start(unitName, unitType)
	}

	// Only handle containers for dependency logic
	if unitType != "container" {
		// For other non-container units, just use the normal restart method
		o.logger.Debug("Direct restart for non-container unit", "unit", unitName, "type", unitType)
		return o.unitManager.Restart(unitName, unitType)
	}

	// For containers, use the dependency-aware restart
	// Parse the unitName to get the service name
	parts := splitUnitName(unitName)
	if len(parts) != 2 {
		// Invalid unit name format, fall back to regular restart
		o.logger.Warn("Invalid unit name format, using direct restart",
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
		o.logger.Debug("Service not found in dependency graph, using direct restart",
			"service", serviceName,
			"project", projectName,
			"error", err)
		return o.unitManager.Restart(unitName, unitType)
	}

	// Always restart the changed service directly - systemd will handle dependency propagation
	o.logger.Debug("Restarting changed service directly - systemd will handle dependency propagation",
		"service", serviceName,
		"project", projectName,
		"dependencies", dependencies)

	return o.unitManager.Restart(unitName, unitType)
}

// waitForUnitsGenerated waits for all specified units to be generated and available in systemd.
// It uses exponential backoff with a maximum timeout to avoid blocking forever.
func (o *DefaultOrchestrator) waitForUnitsGenerated(ctx context.Context, unitNames []string, maxRetries int, initialBackoff time.Duration) error {
	if len(unitNames) == 0 {
		return nil
	}

	o.logger.Debug("Waiting for units to be generated", "count", len(unitNames), "maxRetries", maxRetries)

	backoff := initialBackoff
	lastErr := error(nil)
	userMode := o.configProvider.GetConfig().UserMode

	for attempt := 0; attempt < maxRetries; attempt++ {
		allFound := true
		notFoundUnits := []string{}

		for _, unitName := range unitNames {
			serviceName := unitName + ".service"

			// Attempt to get unit properties to verify it exists
			// This will fail if the unit hasn't been generated yet
			conn, err := o.connectionFactory.NewConnection(ctx, userMode)
			if err != nil {
				o.logger.Debug("Failed to create connection to check unit", "unit", serviceName, "attempt", attempt)
				allFound = false
				notFoundUnits = append(notFoundUnits, serviceName)
				lastErr = err
				continue
			}

			_, err = conn.GetUnitProperties(ctx, serviceName)
			_ = conn.Close() // Always close the connection

			if err != nil {
				allFound = false
				notFoundUnits = append(notFoundUnits, serviceName)
				lastErr = err
				continue
			}
		}

		if allFound {
			o.logger.Debug("All units are now available")
			return nil
		}

		if attempt < maxRetries-1 {
			o.logger.Debug("Units not yet available, retrying",
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"backoff", backoff,
				"notFound", len(notFoundUnits))
			select {
			case <-time.After(backoff):
				// Apply exponential backoff (max 5 seconds per retry)
				backoff = backoff * 2
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
			case <-ctx.Done():
				return fmt.Errorf("unit wait cancelled: %w", ctx.Err())
			}
		}
	}

	// All retries exhausted
	o.logger.Error("Units failed to appear after maximum retries",
		"notFound", len(unitNames),
		"maxRetries", maxRetries,
		"lastError", lastErr)
	return fmt.Errorf("units not generated after %d retries: %w", maxRetries, lastErr)
}

// RestartChangedUnits restarts all changed units in dependency-aware order.
func (o *DefaultOrchestrator) RestartChangedUnits(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	o.logger.Info("Restarting changed units with dependency awareness", "count", len(changedUnits))

	err := o.unitManager.ReloadSystemd()
	if err != nil {
		o.logger.Error("Failed to reload systemd units", "error", err)
		return fmt.Errorf("failed to reload systemd configuration: %w", err)
	}

	// Extract unit names to wait for (container units that will be restarted)
	var containerUnitNames []string
	for _, unit := range changedUnits {
		if unit.Type == "container" {
			containerUnitNames = append(containerUnitNames, unit.Name)
		}
	}

	// Wait for units to be generated with exponential backoff
	if err := o.waitForUnitsGenerated(context.Background(), containerUnitNames, 10, 100*time.Millisecond); err != nil {
		o.logger.Error("Failed waiting for units to be generated", "error", err)
		return fmt.Errorf("units not available after systemd reload: %w", err)
	}

	// Track units with restart failures
	restartFailures := make(map[string]error)

	// First start one-shot services (network, volume, build, image)
	o.logger.Debug("Starting one-shot services first")
	for _, unit := range changedUnits {
		switch unit.Type {
		case "network", "volume", "build", "image":
			unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)

			// For volumes, warn about potential data safety considerations
			if unit.Type == "volume" {
				o.logger.Debug("Starting volume unit - existing data should be preserved",
					"name", unit.Name,
					"note", "Podman typically preserves existing volume data when configuration changes")
			}

			// Use Start() for one-shot services since they don't run continuously
			if err := o.unitManager.Start(unit.Name, unit.Type); err != nil {
				o.logger.Error("Failed to start one-shot service",
					"type", unit.Type,
					"name", unit.Name,
					"error", err)
				restartFailures[unitKey] = err
			} else {
				o.logger.Debug("Successfully started one-shot service", "name", unit.Name, "type", unit.Type)
			}
		}
	}

	// Wait for one-shot services to complete
	time.Sleep(1 * time.Second)

	// Phase 1: Restart container units with dependency awareness
	// On Linux (systemd), use synchronous restarts for proper dependency tracking
	// This ensures the 'restarted' map is updated before checking dependencies
	o.logger.Debug("Restarting container units with dependency awareness")
	restarted := make(map[string]bool)

	for _, unit := range changedUnits {
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
		// Skip one-shot services (already handled) and non-container units
		if unit.Type == "network" || unit.Type == "volume" || unit.Type == "build" || unit.Type == "image" {
			continue
		}

		if unit.Type != "container" {
			// For non-container units, restart synchronously since they're usually quick
			o.logger.Debug("Using direct restart for non-container unit", "name", unit.Name, "type", unit.Type)
			err := o.unitManager.Restart(unit.Name, unit.Type)
			if err != nil {
				o.logger.Error("Failed to restart non-container unit",
					"name", unit.Name, "type", unit.Type, "error", err)
				restartFailures[unitKey] = err
			} else {
				restarted[unitKey] = true
			}
			continue
		}

		// For container units, check dependency graph before restarting
		parts := splitUnitName(unit.Name)
		if len(parts) == 2 {
			projectName := parts[0]
			serviceName := parts[1]

			if dependencyGraph, ok := projectDependencyGraphs[projectName]; ok {
				if isServiceAlreadyRestarted(serviceName, dependencyGraph, projectName, restarted) {
					o.logger.Debug("Skipping restart as unit or its dependent services were already restarted",
						"name", unit.Name, "project", projectName, "service", serviceName)
					continue
				}
			}
		}

		// Restart container synchronously to ensure dependency tracking works correctly
		// Systemd's D-Bus RestartUnit is inherently synchronous and blocks until completion
		o.logger.Debug("Restarting container synchronously", "name", unit.Name)
		err := o.unitManager.Restart(unit.Name, unit.Type)
		if err != nil {
			o.logger.Error("Failed to restart container",
				"name", unit.Name, "error", err)
			restartFailures[unitKey] = err
		} else {
			o.logger.Debug("Unit successfully restarted", "name", unit.Name)
			restarted[unitKey] = true
		}
	}

	// Summarize restart failures if any occurred
	if len(restartFailures) > 0 {
		// Log all failures individually
		for unit, unitErr := range restartFailures {
			o.logger.Error("Unit restart failure", "unit", unit, "error", unitErr)
		}
		o.logger.Error("Some units failed to restart", "count", len(restartFailures))

		// Get the first failing unit for the error message
		firstUnit := ""
		for unit := range restartFailures {
			firstUnit = unit
			break
		}

		return fmt.Errorf("failed to restart %d units. Review logs for details. First error for %s: %v",
			len(restartFailures), firstUnit, restartFailures[firstUnit])
	}

	o.logger.Info("Successfully restarted all changed units", "count", len(changedUnits))
	return nil
}

// initiateAsyncRestart starts a unit restart without waiting for completion.
// NOTE: Currently unused - kept for potential future async restart needs (e.g., macOS compatibility layer).
// Linux systemd operations use synchronous RestartUnit for proper dependency tracking.
func (o *DefaultOrchestrator) initiateAsyncRestart(unitName, _ string) error { //nolint:unused // Kept for potential future async restart needs
	conn, err := o.connectionFactory.NewConnection(context.Background(), o.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := unitName + ".service"
	o.logger.Debug("Initiating async restart", "name", serviceName)

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
		o.logger.Debug("Restart initiation result", "name", serviceName, "result", result)
	}()

	return nil
}

// checkUnitFinalStatus checks if a unit has reached active state, with handling for activating states.
// NOTE: Currently unused - kept for potential future async restart needs (e.g., macOS compatibility layer).
// Linux systemd operations use synchronous RestartUnit which blocks until completion.
func (o *DefaultOrchestrator) checkUnitFinalStatus(unitName, _ string) error { //nolint:unused // Kept for potential future async restart needs
	conn, err := o.connectionFactory.NewConnection(context.Background(), o.configProvider.GetConfig().UserMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := unitName + ".service"
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

		o.logger.Debug("Unit still activating, waiting for completion",
			"name", serviceName, "subState", subStateStr)

		// Wait based on sub-state
		waitTime := o.configProvider.GetConfig().UnitStartTimeout
		if subStateStr == "start" {
			waitTime = o.configProvider.GetConfig().ImagePullTimeout
		}

		time.Sleep(waitTime)

		// Check final state
		finalActiveState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
		if err == nil {
			finalState := finalActiveState.Value.Value().(string)
			if finalState == "active" {
				o.logger.Info("Unit successfully reached active state", "name", serviceName)
				return nil
			}
			currentState = finalState
		}
	}

	// If we get here, the unit is not active
	details := o.unitManager.GetUnitFailureDetails(serviceName)
	return fmt.Errorf("unit failed to reach active state: %s%s", currentState, details)
}
