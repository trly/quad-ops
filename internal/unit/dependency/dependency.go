// Package dependency handles service dependency management
package dependency

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/unit/model"
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
func ApplyDependencyRelationships(unit *model.QuadletUnitConfig, serviceName string, dependencies map[string]*ServiceDependency, projectName string) {
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

	// For container units, add dependencies on attached networks and volumes
	if unit.Type == "container" {
		// Add dependencies on networks
		for _, networkRef := range unit.Container.Network {
			// Only add dependency if it's a project-defined network (has .network suffix)
			if strings.HasSuffix(networkRef, ".network") {
				// Use network name for dependency, not the filename
				networkName := strings.TrimSuffix(networkRef, ".network")
				unit.Systemd.After = append(unit.Systemd.After, networkName+"-network.service")
				unit.Systemd.Requires = append(unit.Systemd.Requires, networkName+"-network.service")
			}
		}

		// Add dependencies on volumes
		for _, volumeRef := range unit.Container.Volume {
			// Extract volume reference (before the colon)
			parts := strings.Split(volumeRef, ":")
			if len(parts) > 0 {
				volumeName := parts[0]
				// Only add dependency if it's a project-defined volume (has .volume suffix)
				if strings.HasSuffix(volumeName, ".volume") {
					// Use volume name for dependency, not the filename
					volBaseName := strings.TrimSuffix(volumeName, ".volume")
					unit.Systemd.After = append(unit.Systemd.After, volBaseName+"-volume.service")
					unit.Systemd.Requires = append(unit.Systemd.Requires, volBaseName+"-volume.service")
				}
			}
		}
	}
}

// FindTopLevelDependentService finds the top-most (leaf) service that depends on this service.
func FindTopLevelDependentService(serviceName string, dependencyTree map[string]*ServiceDependency) string {
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
	higherDep := FindTopLevelDependentService(dependentService, dependencyTree)
	if higherDep != "" {
		return higherDep
	}

	// This is the top-level dependent service
	return dependentService
}
