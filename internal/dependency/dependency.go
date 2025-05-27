// Package dependency provides service dependency graph management for Docker Compose projects.
package dependency

import (
	"fmt"

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
