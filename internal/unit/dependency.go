package unit

import (
	"fmt"
	"strings"

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

		// Special handling for build dependencies
		// If the dependency name ends with -build, it's a build unit
		formattedDepName := ""
		if strings.HasSuffix(depName, "-build") {
			// Build units have their service name with an additional -build suffix
			// from Quadlet, so we need to adjust the service name accordingly
			formattedDepName = fmt.Sprintf("%s-build.service", depPrefixedName)
		} else {
			// Regular container unit
			formattedDepName = fmt.Sprintf("%s.service", depPrefixedName)
		}

		// Add dependency to After and Requires lists
		unit.Systemd.After = append(unit.Systemd.After, formattedDepName)
		unit.Systemd.Requires = append(unit.Systemd.Requires, formattedDepName)
	}

	// Skip PartOf relationships to avoid circular dependencies.
	// Docker Compose depends_on relationships already establish proper startup order
	// via After/Requires directives. Adding PartOf creates circular dependencies
	// when Service A requires Service B, but Service B is also "part of" Service A.
	// The dependency-aware restart logic in restart.go handles service restarts
	// without needing PartOf directives.

	// For container units, add dependencies on attached networks and volumes
	if unit.Type == "container" {
		// Add dependencies on networks
		for _, networkRef := range unit.Container.Network {
			// Only add dependency if it's a project-defined network (has .network suffix)
			if strings.HasSuffix(networkRef, ".network") {
				// Convert to service name by replacing .network with -network.service
				networkServiceName := strings.Replace(networkRef, ".network", "-network.service", 1)
				unit.Systemd.After = append(unit.Systemd.After, networkServiceName)
				unit.Systemd.Requires = append(unit.Systemd.Requires, networkServiceName)
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
					// Convert to service name by replacing .volume with -volume.service
					volumeServiceName := strings.Replace(volumeName, ".volume", "-volume.service", 1)
					unit.Systemd.After = append(unit.Systemd.After, volumeServiceName)
					unit.Systemd.Requires = append(unit.Systemd.Requires, volumeServiceName)
				}
			}
		}
	}
}
