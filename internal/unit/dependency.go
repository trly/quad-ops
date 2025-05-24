package unit

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/dominikbraun/graph"
)

// ServiceDependencyGraph represents the dependency relationships between services using a directed graph.
type ServiceDependencyGraph struct {
	graph graph.Graph[string, string]
}

// NewServiceDependencyGraph creates a new service dependency graph.
func NewServiceDependencyGraph() *ServiceDependencyGraph {
	// Create a directed acyclic graph with string vertices
	g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())
	return &ServiceDependencyGraph{
		graph: g,
	}
}

// AddService adds a service to the dependency graph.
func (sdg *ServiceDependencyGraph) AddService(serviceName string) error {
	return sdg.graph.AddVertex(serviceName)
}

// AddDependency adds a dependency relationship where dependent depends on dependency.
func (sdg *ServiceDependencyGraph) AddDependency(dependent, dependency string) error {
	return sdg.graph.AddEdge(dependency, dependent)
}

// GetDependencies returns the services that the given service depends on.
func (sdg *ServiceDependencyGraph) GetDependencies(serviceName string) ([]string, error) {
	predecessors, err := sdg.graph.PredecessorMap()
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0, len(predecessors[serviceName]))
	for dep := range predecessors[serviceName] {
		deps = append(deps, dep)
	}

	return deps, nil
}

// GetDependents returns the services that depend on the given service.
func (sdg *ServiceDependencyGraph) GetDependents(serviceName string) ([]string, error) {
	successors, err := sdg.graph.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0, len(successors[serviceName]))
	for dep := range successors[serviceName] {
		deps = append(deps, dep)
	}

	return deps, nil
}

// GetTopologicalOrder returns services in topological order (dependencies first).
func (sdg *ServiceDependencyGraph) GetTopologicalOrder() ([]string, error) {
	return graph.TopologicalSort(sdg.graph)
}

// HasCycles checks if the dependency graph contains cycles.
func (sdg *ServiceDependencyGraph) HasCycles() bool {
	// Since we use graph.Acyclic(), cycle creation is prevented at edge addition time
	// But we can still check for cycles if needed
	_, err := graph.TopologicalSort(sdg.graph)
	return err != nil
}

// BuildServiceDependencyGraph builds a dependency graph for all services in a project.
func BuildServiceDependencyGraph(project *types.Project) (*ServiceDependencyGraph, error) {
	sdg := NewServiceDependencyGraph()

	// Add all services as vertices first
	for serviceName := range project.Services {
		if err := sdg.AddService(serviceName); err != nil {
			return nil, fmt.Errorf("failed to add service %s: %w", serviceName, err)
		}
	}

	// Add dependency edges based on depends_on relationships
	for serviceName, service := range project.Services {
		for depName := range service.DependsOn {
			if err := sdg.AddDependency(serviceName, depName); err != nil {
				return nil, fmt.Errorf("failed to add dependency %s -> %s: %w", serviceName, depName, err)
			}
		}
	}

	return sdg, nil
}

// ApplyDependencyRelationships applies dependencies to a quadlet unit based on the dependency graph.
func ApplyDependencyRelationships(unit *QuadletUnit, serviceName string, dependencyGraph *ServiceDependencyGraph, projectName string) error {
	// Get dependencies for this service
	dependencies, err := dependencyGraph.GetDependencies(serviceName)
	if err != nil {
		return fmt.Errorf("failed to get dependencies for service %s: %w", serviceName, err)
	}

	// Apply regular dependencies (services this one depends on)
	for _, depName := range dependencies {
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

	return nil
}
