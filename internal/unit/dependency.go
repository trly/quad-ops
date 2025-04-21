package unit

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// ServiceDependency represents the dependencies of a service in both directions.
type ServiceDependency struct {
	// Dependencies is the list of services this service depends on
	Dependencies map[string]struct{}
	// DependentServices is the list of services that depend on this service
	DependentServices map[string]struct{}
}

// BuildServiceDependencyTree builds a bidirectional dependency tree for all services in a project.
func BuildServiceDependencyTree(project *types.Project) map[string]*ServiceDependency {
	dependencies := make(map[string]*ServiceDependency)

	// Initialize the dependency tree for all services
	for serviceName := range project.Services {
		dependencies[serviceName] = &ServiceDependency{
			Dependencies:      make(map[string]struct{}),
			DependentServices: make(map[string]struct{}),
		}
	}

	// Populate the dependency tree based on depends_on relationships
	for serviceName, service := range project.Services {
		for depName := range service.DependsOn {
			// This service depends on depName
			dependencies[serviceName].Dependencies[depName] = struct{}{}
			// depName has this service as a dependent
			dependencies[depName].DependentServices[serviceName] = struct{}{}
		}
	}

	return dependencies
}

// ApplyDependencyRelationships applies both regular dependencies (After/Requires) and reverse
// dependencies (PartOf) to a quadlet unit based on the dependency tree.
func ApplyDependencyRelationships(unit *QuadletUnit, serviceName string, dependencies map[string]*ServiceDependency, projectName string) { //nolint:whitespace // False positive

	// Apply regular dependencies (services this one depends on)
	for depName := range dependencies[serviceName].Dependencies {
		depPrefixedName := fmt.Sprintf("%s-%s", projectName, depName)
		formattedDepName := fmt.Sprintf("%s.service", depPrefixedName)

		// Add dependency to After and Requires lists
		unit.Systemd.After = append(unit.Systemd.After, formattedDepName)
		unit.Systemd.Requires = append(unit.Systemd.Requires, formattedDepName)
	}

	// Apply reverse dependencies (services that depend on this one)
	for dependentService := range dependencies[serviceName].DependentServices {
		dependentPrefixedName := fmt.Sprintf("%s-%s", projectName, dependentService)
		formattedDependentName := fmt.Sprintf("%s.service", dependentPrefixedName)

		// Add PartOf directive to make this service restart when dependent services restart
		unit.Systemd.PartOf = append(unit.Systemd.PartOf, formattedDependentName)
	}
}
