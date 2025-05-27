package systemd

import (
	"context"
	"fmt"
	"time"

	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/log"
)

// OrchestrationResult represents the result of an orchestration operation.
type OrchestrationResult struct {
	Success bool
	Errors  map[string]error
}

// UnitChange represents a unit that has changed and needs to be restarted.
type UnitChange struct {
	Name string
	Type string
	Unit Unit
}

// StartUnitDependencyAware starts or restarts a unit while being dependency-aware.
func StartUnitDependencyAware(unitName string, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error {
	log.GetLogger().Debug("Starting/restarting unit with dependency awareness", "unit", unitName, "type", unitType)

	// Handle different unit types appropriately
	switch unitType {
	case "build", "image", "network", "volume":
		// For one-shot services, use Start() instead of Restart()
		log.GetLogger().Debug("Starting one-shot service", "unit", unitName, "type", unitType)
		unit := NewBaseUnit(unitName, unitType)
		return unit.Start()
	}

	// Only handle containers for dependency logic
	if unitType != "container" {
		// For other non-container units, just use the normal restart method
		log.GetLogger().Debug("Direct restart for non-container unit", "unit", unitName, "type", unitType)
		unit := NewBaseUnit(unitName, unitType)
		return unit.Restart()
	}

	// For containers, use the dependency-aware restart
	// Parse the unitName to get the service name
	// Example: project-service -> service
	parts := splitUnitName(unitName)
	if len(parts) != 2 {
		// Invalid unit name format, fall back to regular restart
		log.GetLogger().Warn("Invalid unit name format, using direct restart",
			"unit", unitName,
			"expected", "project-service")
		unit := NewBaseUnit(unitName, unitType)
		return unit.Restart()
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
		unit := NewBaseUnit(unitName, unitType)
		return unit.Restart()
	}

	// Always restart the changed service directly - systemd will handle dependency propagation
	// The systemd After/Requires directives ensure proper restart order automatically
	log.GetLogger().Debug("Restarting changed service directly - systemd will handle dependency propagation",
		"service", serviceName,
		"project", projectName,
		"dependencies", dependencies)

	unit := NewBaseUnit(unitName, unitType)
	return unit.Restart()
}

// RestartChangedUnits restarts all changed units in dependency-aware order.
func RestartChangedUnits(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	log.GetLogger().Info("Restarting changed units with dependency awareness", "count", len(changedUnits))
	err := ReloadSystemd()
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
		err := initiateAsyncRestart(unit.Unit)
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
			err := checkUnitFinalStatus(unit.Unit)
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

// splitUnitName splits a unit name like "project-service" into ["project", "service"].
func splitUnitName(unitName string) []string {
	// Find the last dash in the name
	for i := len(unitName) - 1; i >= 0; i-- {
		if unitName[i] == '-' {
			return []string{unitName[:i], unitName[i+1:]}
		}
	}
	return []string{unitName}
}

// isServiceAlreadyRestarted checks if the service itself is already restarted
// or if any services that would cause this service to restart are already restarted.
func isServiceAlreadyRestarted(serviceName string, dependencyGraph *dependency.ServiceDependencyGraph, projectName string, restarted map[string]bool) bool {
	// Check if this service is already restarted
	unitKey := fmt.Sprintf("%s-%s.container", projectName, serviceName)
	if restarted[unitKey] {
		return true
	}

	// For each service this one depends on, check if it's been restarted
	// We only check dependencies because a change in a dependency causes us to restart
	// (due to After/Requires). Changes in dependent services don't affect us.
	dependencies, err := dependencyGraph.GetDependencies(serviceName)
	if err == nil {
		for _, dep := range dependencies {
			if restarted[fmt.Sprintf("%s-%s.container", projectName, dep)] {
				return true
			}
		}
	}

	return false
}

// initiateAsyncRestart starts a unit restart without waiting for completion.
func initiateAsyncRestart(unit Unit) error {
	conn, err := GetSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := unit.GetServiceName()
	log.GetLogger().Debug("Initiating async restart", "name", serviceName)

	// Check if unit is loaded before attempting restart
	loadState, err := conn.GetUnitPropertyContext(GetContext(), serviceName, "LoadState")
	if err != nil {
		return fmt.Errorf("error checking unit load state %s: %w", serviceName, err)
	}

	if loadState.Value.Value().(string) != "loaded" {
		return fmt.Errorf("unit %s is not loaded (LoadState: %s), cannot restart", serviceName, loadState.Value.Value().(string))
	}

	// Initiate restart without waiting for completion
	ch := make(chan string)
	_, err = conn.RestartUnitContext(context.Background(), serviceName, "replace", ch)
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
func checkUnitFinalStatus(unit Unit) error {
	conn, err := GetSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer conn.Close()

	serviceName := unit.GetServiceName()

	// Check current state
	activeState, err := conn.GetUnitPropertyContext(GetContext(), serviceName, "ActiveState")
	if err != nil {
		return fmt.Errorf("error getting unit state %s: %w", serviceName, err)
	}

	currentState := activeState.Value.Value().(string)
	if currentState == "active" {
		return nil // Already active, success
	}

	// If it's activating, wait and check again
	if currentState == "activating" {
		subState, err := conn.GetUnitPropertyContext(GetContext(), serviceName, "SubState")
		subStateStr := "unknown"
		if err == nil {
			subStateStr = subState.Value.Value().(string)
		}

		log.GetLogger().Debug("Unit still activating, waiting for completion",
			"name", serviceName, "subState", subStateStr)

		// Wait based on sub-state
		waitTime := 10 * time.Second
		if subStateStr == "start" {
			waitTime = 15 * time.Second // Even more time for startup/image pulls
		}

		time.Sleep(waitTime)

		// Check final state
		finalActiveState, err := conn.GetUnitPropertyContext(GetContext(), serviceName, "ActiveState")
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
	details := GetUnitFailureDetails(serviceName)
	return fmt.Errorf("unit failed to reach active state: %s%s", currentState, details)
}
