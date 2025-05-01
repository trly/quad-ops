package unit

import (
	"fmt"
	"log"
	"time"

	"github.com/trly/quad-ops/internal/config"
)

// Change tracks changes to a unit.
type Change struct {
	Name string
	Type string
	Hash []byte
}

// StartUnitDependencyAware starts or restarts a unit while being dependency-aware.
func StartUnitDependencyAware(unitName string, unitType string, dependencyTree map[string]*ServiceDependency) error {
	if config.GetConfig().Verbose {
		log.Printf("Starting/restarting unit %s.%s with dependency awareness", unitName, unitType)
	}

	// For network and volume units, start them first before containers
	if unitType == "network" || unitType == "volume" {
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Only handle containers for dependency logic
	if unitType != "container" {
		// For other non-container units, just use the normal restart method
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// For containers, use the dependency-aware restart
	// Parse the unitName to get the service name
	// Example: project-service -> service
	parts := splitUnitName(unitName)
	if len(parts) != 2 {
		// Invalid unit name format, fall back to regular restart
		log.Printf("Invalid unit name format: %s", unitName)
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	projectName := parts[0]
	serviceName := parts[1]

	// Check if this service is in the dependency tree
	if _, ok := dependencyTree[serviceName]; !ok {
		// Service not in dependency tree, fall back to regular restart
		if config.GetConfig().Verbose {
			log.Printf("Service %s not found in dependency tree, using normal restart", serviceName)
		}
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Find the top-level dependent service that depends on this one (if any)
	dependent := findTopLevelDependentService(serviceName, dependencyTree)

	// If no dependent service, just restart this one
	if dependent == "" {
		if config.GetConfig().Verbose {
			log.Printf("No dependent services found for %s, restarting directly", serviceName)
		}
		unit := &BaseSystemdUnit{Name: unitName, Type: unitType}
		return unit.Restart()
	}

	// Found a top-level dependent service, restart that instead
	if config.GetConfig().Verbose {
		log.Printf("Found top-level dependent service %s for %s, restarting that instead",
			dependent, serviceName)
	}

	// Format the systemd unit service name correctly
	dependentUnitName := fmt.Sprintf("%s-%s", projectName, dependent)
	unit := &BaseSystemdUnit{Name: dependentUnitName, Type: unitType}
	return unit.Restart()
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
	err := ReloadSystemd()
	if err != nil {
		log.Printf("error reloading systemd units: %v", err)
		return err
	}

	// Wait for systemd to process the changes
	time.Sleep(2 * time.Second)

	// First restart network and volume units
	for _, unit := range changedUnits {
		if unit.Type == "network" || unit.Type == "volume" {
			systemdUnit := unit.GetSystemdUnit()
			if err := systemdUnit.Restart(); err != nil {
				log.Printf("error restarting %s unit %s: %v", unit.Type, unit.Name, err)
			} else if config.GetConfig().Verbose {
				log.Printf("successfully restarted unit %s.service", unit.Name)
			}
		}
	}

	// Wait for networks and volumes to be fully available
	time.Sleep(1 * time.Second)

	// Track units that have been restarted
	restarted := make(map[string]bool)

	for _, unit := range changedUnits {
		// Skip if already restarted or if it's a network or volume (handled earlier)
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
		if restarted[unitKey] || unit.Type == "network" || unit.Type == "volume" {
			continue
		}

		// For non-container units or if we don't have dependency trees, use normal restart
		if unit.Type != "container" || len(projectDependencyTrees) == 0 {
			// Use the regular restart method
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.Printf("error restarting unit %s: %v", unit.Name, err)
			}
			restarted[unitKey] = true
			continue
		}

		// For container units, try to find the project name and dependency tree
		parts := splitUnitName(unit.Name)
		if len(parts) != 2 {
			// Invalid unit name format, fall back to regular restart
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.Printf("error restarting unit %s: %v", unit.Name, err)
			}
			restarted[unitKey] = true
			continue
		}

		projectName := parts[0]
		serviceName := parts[1]

		// Find the dependency tree for this project
		dependencyTree, ok := projectDependencyTrees[projectName]
		if !ok {
			// No dependency tree for this project, use normal restart
			systemdUnit := unit.GetSystemdUnit()
			err := systemdUnit.Restart()
			if err != nil {
				log.Printf("error restarting unit %s: %v", unit.Name, err)
			}
			restarted[unitKey] = true
			continue
		}

		// Skip if this service or any dependent service has already been restarted
		if isServiceAlreadyRestarted(serviceName, dependencyTree, projectName, restarted) {
			if config.GetConfig().Verbose {
				log.Printf("skipping restart of %s as it or its dependent services were already restarted", unit.Name)
			}
			continue
		}

		// Use dependency-aware restart
		err := StartUnitDependencyAware(unit.Name, unit.Type, dependencyTree)
		if err != nil {
			log.Printf("error restarting unit %s: %v", unit.Name, err)
		}

		// Mark all dependent services as restarted since they will be restarted by systemd
		markDependentsAsRestarted(serviceName, dependencyTree, projectName, restarted)
	}

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
