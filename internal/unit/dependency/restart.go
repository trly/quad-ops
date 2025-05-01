package dependency

import (
	"fmt"
)

// SplitUnitName splits a unit name like "project-service" into ["project", "service"].
func SplitUnitName(unitName string) []string {
	// Find the last dash in the name
	for i := len(unitName) - 1; i >= 0; i-- {
		if unitName[i] == '-' {
			return []string{unitName[:i], unitName[i+1:]}
		}
	}
	return []string{unitName}
}

// IsServiceAlreadyRestarted checks if the service itself is already restarted
// or if any services that would cause this service to restart are already restarted.
func IsServiceAlreadyRestarted(serviceName string, dependencyTree map[string]*ServiceDependency, projectName string, restarted map[string]bool) bool {
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

// MarkDependentsAsRestarted marks all services that depend on the given service as restarted.
func MarkDependentsAsRestarted(serviceName string, dependencyTree map[string]*ServiceDependency, projectName string, restarted map[string]bool) {
	// Mark this service as restarted
	unitKey := fmt.Sprintf("%s-%s.container", projectName, serviceName)
	restarted[unitKey] = true

	// Mark all services that depend on this one as restarted
	for dependent := range dependencyTree[serviceName].DependentServices {
		// Skip if already marked
		dependentKey := fmt.Sprintf("%s-%s.container", projectName, dependent)
		if !restarted[dependentKey] {
			// Recursively mark all dependent services
			MarkDependentsAsRestarted(dependent, dependencyTree, projectName, restarted)
		}
	}
}