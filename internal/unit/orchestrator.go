package unit

import (
	"fmt"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/log"
)

// Change tracks changes to a unit.
type Change struct {
	Name string
	Type string
	Hash []byte
}

// StartUnitDependencyAware starts or restarts a unit while being dependency-aware.
func StartUnitDependencyAware(unitName string, unitType string, dependencyTree map[string]*ServiceDependency) error {
	log.GetLogger().Debug("Starting/restarting unit with dependency awareness", "unit", unitName, "type", unitType)

	// For network and volume units, start them first before containers
	if unitType == "network" || unitType == "volume" {
		log.GetLogger().Debug("Direct restart for infrastructure unit", "unit", unitName, "type", unitType)
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Only handle containers for dependency logic
	if unitType != "container" {
		// For other non-container units, just use the normal restart method
		log.GetLogger().Debug("Direct restart for non-container unit", "unit", unitName, "type", unitType)
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
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
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	projectName := parts[0]
	serviceName := parts[1]

	// Check if this service is in the dependency tree
	if _, ok := dependencyTree[serviceName]; !ok {
		// Service not in dependency tree, fall back to regular restart
		log.GetLogger().Debug("Service not found in dependency tree, using direct restart",
			"service", serviceName,
			"project", projectName,
			"availableServices", getDependencyTreeKeys(dependencyTree))
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Find the top-level dependent service that depends on this one (if any)
	dependent := findTopLevelDependentService(serviceName, dependencyTree)

	// If no dependent service, just restart this one
	if dependent == "" {
		log.GetLogger().Debug("No dependent services found, restarting directly",
			"service", serviceName,
			"project", projectName,
			"dependencies", getDependencyListForService(serviceName, dependencyTree))
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Found a top-level dependent service, restart that instead
	log.GetLogger().Debug("Found top-level dependent service, restarting that instead",
		"dependent", dependent,
		"service", serviceName,
		"project", projectName,
		"dependencyChain", getDependencyChain(serviceName, dependent, dependencyTree))

	// Format the systemd unit service name correctly
	dependentUnitName := fmt.Sprintf("%s-%s", projectName, dependent)
	unit := &BaseSystemdUnit{Name: dependentUnitName, Type: unitType}
	return unit.Restart()
}

// getDependencyTreeKeys returns a list of all service names in the dependency tree for logging.
func getDependencyTreeKeys(dependencyTree map[string]*ServiceDependency) []string {
	keys := make([]string, 0, len(dependencyTree))
	for key := range dependencyTree {
		keys = append(keys, key)
	}
	return keys
}

// getDependencyListForService returns a list of dependencies for a service for logging.
func getDependencyListForService(serviceName string, dependencyTree map[string]*ServiceDependency) []string {
	deps := make([]string, 0)
	if depEntry, ok := dependencyTree[serviceName]; ok {
		for dep := range depEntry.Dependencies {
			deps = append(deps, dep)
		}
		for dep := range depEntry.DependentServices {
			deps = append(deps, fmt.Sprintf("(dependent) %s", dep))
		}
	}
	return deps
}

// getDependencyChain returns a description of the dependency chain between two services.
func getDependencyChain(serviceName, dependentService string, dependencyTree map[string]*ServiceDependency) string {
	chain := []string{}
	current := serviceName
	for current != dependentService {
		next := ""
		for dep := range dependencyTree[current].DependentServices {
			if isInDependencyPath(dep, dependentService, dependencyTree, make(map[string]bool)) {
				next = dep
				break
			}
		}
		if next == "" {
			break
		}
		chain = append(chain, fmt.Sprintf("%s → %s", current, next))
		current = next
	}
	return strings.Join(chain, " → ")
}

// isInDependencyPath checks if target is in the dependency path of source.
func isInDependencyPath(source, target string, dependencyTree map[string]*ServiceDependency, visited map[string]bool) bool {
	if source == target {
		return true
	}
	if visited[source] {
		return false
	}
	visited[source] = true

	for dep := range dependencyTree[source].DependentServices {
		if isInDependencyPath(dep, target, dependencyTree, visited) {
			return true
		}
	}
	return false
}

// findTopLevelDependentService finds the top-most (leaf) service that depends on this service.
func findTopLevelDependentService(serviceName string, dependencyTree map[string]*ServiceDependency) string {
	// If no dependent services, return empty string
	if len(dependencyTree[serviceName].DependentServices) == 0 {
		return ""
	}

	// Get one of the dependent services
	var dependentService string
	for dep := range dependencyTree[serviceName].DependentServices {
		dependentService = dep
		break
	}

	// Recursively find the top-level dependent service
	higherDep := findTopLevelDependentService(dependentService, dependencyTree)
	if higherDep != "" {
		return higherDep
	}

	// This is the top-level dependent service
	return dependentService
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

// RestartChangedUnits restarts all changed units in dependency-aware order.
func RestartChangedUnits(changedUnits []QuadletUnit, projectDependencyTrees map[string]map[string]*ServiceDependency) error {
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

	// First restart network and volume units
	log.GetLogger().Debug("Restarting network and volume units first")
	for _, unit := range changedUnits {
		if unit.Type == "network" || unit.Type == "volume" {
			systemdUnit := unit.GetSystemdUnit()
			unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
			if err := systemdUnit.Restart(); err != nil {
				log.GetLogger().Error("Failed to restart unit",
					"type", unit.Type,
					"name", unit.Name,
					"error", err)
				restartFailures[unitKey] = err
			} else {
				log.GetLogger().Debug("Successfully restarted unit", "name", unit.Name, "type", unit.Type)
			}
		}
	}

	// Wait for networks and volumes to be fully available
	time.Sleep(1 * time.Second)

	// Track units that have been restarted
	restarted := make(map[string]bool)

	log.GetLogger().Debug("Restarting container units with dependency awareness")
	for _, unit := range changedUnits {
		// Skip if already restarted or if it's a network or volume (handled earlier)
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
		if restarted[unitKey] || unit.Type == "network" || unit.Type == "volume" {
			continue
		}

		// For non-container units or if we don't have dependency trees, use normal restart
		if unit.Type != "container" || len(projectDependencyTrees) == 0 {
			log.GetLogger().Debug("Using direct restart for unit", "name", unit.Name, "type", unit.Type)
			// Use the regular restart method
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.GetLogger().Error("Failed to restart unit",
					"name", unit.Name,
					"type", unit.Type,
					"error", err)
				restartFailures[unitKey] = err
			}
			restarted[unitKey] = true
			continue
		}

		// For container units, try to find the project name and dependency tree
		parts := splitUnitName(unit.Name)
		if len(parts) != 2 {
			log.GetLogger().Debug("Invalid unit name format, using direct restart", "name", unit.Name)
			// Invalid unit name format, fall back to regular restart
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.GetLogger().Error("Failed to restart unit with invalid name format",
					"name", unit.Name,
					"error", err)
				restartFailures[unitKey] = err
			}
			restarted[unitKey] = true
			continue
		}

		projectName := parts[0]
		serviceName := parts[1]

		// Find the dependency tree for this project
		dependencyTree, ok := projectDependencyTrees[projectName]
		if !ok {
			log.GetLogger().Debug("No dependency tree found for project, using direct restart",
				"project", projectName,
				"service", serviceName)
			// No dependency tree for this project, use normal restart
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.GetLogger().Error("Failed to restart unit (no dependency tree)",
					"name", unit.Name,
					"error", err)
				restartFailures[unitKey] = err
			}
			restarted[unitKey] = true
			continue
		}

		// Skip if this service or any dependent service has already been restarted
		if isServiceAlreadyRestarted(serviceName, dependencyTree, projectName, restarted) {
			log.GetLogger().Debug("Skipping restart as unit or its dependent services were already restarted",
				"name", unit.Name,
				"project", projectName,
				"service", serviceName)
			continue
		}

		// Use dependency-aware restart
		log.GetLogger().Debug("Using dependency-aware restart",
			"name", unit.Name,
			"project", projectName,
			"service", serviceName)
		err := StartUnitDependencyAware(unit.Name, unit.Type, dependencyTree)
		if err != nil {
			log.GetLogger().Error("Failed to restart unit with dependency awareness",
				"name", unit.Name,
				"project", projectName,
				"service", serviceName,
				"error", err)
			restartFailures[unitKey] = err
		}

		// Mark all dependent services as restarted since they will be restarted by systemd
		markDependentsAsRestarted(serviceName, dependencyTree, projectName, restarted)
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

// markDependentsAsRestarted marks all services that depend on the given service as restarted.
func markDependentsAsRestarted(serviceName string, dependencyTree map[string]*ServiceDependency, projectName string, restarted map[string]bool) { //nolint:whitespace // False positive

	// Mark this service as restarted
	unitKey := fmt.Sprintf("%s-%s.container", projectName, serviceName)
	restarted[unitKey] = true

	// Mark all services that depend on this one as restarted
	for dependent := range dependencyTree[serviceName].DependentServices {
		// Skip if already marked
		dependentKey := fmt.Sprintf("%s-%s.container", projectName, dependent)
		if !restarted[dependentKey] {
			// Recursively mark all dependent services
			markDependentsAsRestarted(dependent, dependencyTree, projectName, restarted)
		}
	}
}

// isServiceAlreadyRestarted checks if the service itself is already restarted
// or if any services that would cause this service to restart are already restarted.
func isServiceAlreadyRestarted(serviceName string, dependencyTree map[string]*ServiceDependency, projectName string, restarted map[string]bool) bool {
	// Check if this service is already restarted
	unitKey := fmt.Sprintf("%s-%s.container", projectName, serviceName)
	if restarted[unitKey] {
		return true
	}

	// For each service this one depends on, check if it's been restarted
	// We only check dependencies because a change in a dependency causes us to restart
	// (due to After/Requires). Changes in dependent services don't affect us.
	for dep := range dependencyTree[serviceName].Dependencies {
		if restarted[fmt.Sprintf("%s-%s.container", projectName, dep)] {
			return true
		}
	}

	return false
}
