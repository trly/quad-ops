package systemd

import (
	"fmt"

	"github.com/trly/quad-ops/internal/dependency"
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

// These functions have been moved to the DefaultOrchestrator implementation in providers.go
// and are kept here only for any potential backward compatibility needs.
