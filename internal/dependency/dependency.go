// Package dependency provides service dependency graph management for Docker Compose projects.
package dependency

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/compose-spec/compose-go/v2/types"
)

// ServiceDependencyGraph models dependencies between services using adjacency maps.
// Edge direction: dependency -> dependent (i.e., B -> A means A depends on B).
type ServiceDependencyGraph struct {
	mu   sync.RWMutex                   // protects concurrent access to maps
	succ map[string]map[string]struct{} // node -> set of successors (dependents)
	pred map[string]map[string]struct{} // node -> set of predecessors (dependencies)
}

// NewServiceDependencyGraph creates a new, empty dependency graph.
func NewServiceDependencyGraph() *ServiceDependencyGraph {
	return &ServiceDependencyGraph{
		succ: make(map[string]map[string]struct{}),
		pred: make(map[string]map[string]struct{}),
	}
}

// AddService ensures a service exists in the graph.
func (sdg *ServiceDependencyGraph) AddService(serviceName string) error {
	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	sdg.mu.Lock()
	defer sdg.mu.Unlock()

	if _, ok := sdg.succ[serviceName]; !ok {
		sdg.succ[serviceName] = make(map[string]struct{})
	}
	if _, ok := sdg.pred[serviceName]; !ok {
		sdg.pred[serviceName] = make(map[string]struct{})
	}
	return nil
}

// AddDependency adds a dependency relationship where `dependent` depends on `dependency`.
// This creates an edge: dependency -> dependent.
func (sdg *ServiceDependencyGraph) AddDependency(dependent, dependency string) error {
	if dependent == "" || dependency == "" {
		return fmt.Errorf("dependent and dependency must be non-empty")
	}
	if dependent == dependency {
		return fmt.Errorf("self-dependency is not allowed: %s", dependent)
	}

	sdg.mu.Lock()
	defer sdg.mu.Unlock()

	// Ensure vertices exist
	if _, ok := sdg.succ[dependent]; !ok {
		sdg.succ[dependent] = make(map[string]struct{})
	}
	if _, ok := sdg.pred[dependent]; !ok {
		sdg.pred[dependent] = make(map[string]struct{})
	}
	if _, ok := sdg.succ[dependency]; !ok {
		sdg.succ[dependency] = make(map[string]struct{})
	}
	if _, ok := sdg.pred[dependency]; !ok {
		sdg.pred[dependency] = make(map[string]struct{})
	}

	// Add edge if not present
	if _, ok := sdg.succ[dependency][dependent]; ok {
		return nil
	}
	sdg.succ[dependency][dependent] = struct{}{}
	sdg.pred[dependent][dependency] = struct{}{}
	return nil
}

// GetDependencies returns the services that the given service depends on.
func (sdg *ServiceDependencyGraph) GetDependencies(serviceName string) ([]string, error) {
	sdg.mu.RLock()
	defer sdg.mu.RUnlock()

	preds, ok := sdg.pred[serviceName]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}
	deps := make([]string, 0, len(preds))
	for dep := range preds {
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	return deps, nil
}

// GetDependents returns the services that depend on the given service.
func (sdg *ServiceDependencyGraph) GetDependents(serviceName string) ([]string, error) {
	sdg.mu.RLock()
	defer sdg.mu.RUnlock()

	succs, ok := sdg.succ[serviceName]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}
	deps := make([]string, 0, len(succs))
	for dep := range succs {
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	return deps, nil
}

// GetTopologicalOrder returns services in topological order (dependencies first).
// Kahn's algorithm with deterministic tie-breaking (lexical).
func (sdg *ServiceDependencyGraph) GetTopologicalOrder() ([]string, error) {
	sdg.mu.RLock()

	// Build indegree map - copy the data we need while holding the lock
	indeg := make(map[string]int, len(sdg.pred))
	for v := range sdg.pred {
		indeg[v] = len(sdg.pred[v])
	}

	// Copy successor relationships we'll need
	succCopy := make(map[string]map[string]struct{}, len(sdg.succ))
	for v, succs := range sdg.succ {
		succCopy[v] = make(map[string]struct{}, len(succs))
		for w := range succs {
			succCopy[v][w] = struct{}{}
		}
	}

	sdg.mu.RUnlock()

	// Initialize zero-indegree set
	zero := make([]string, 0)
	for v, d := range indeg {
		if d == 0 {
			zero = append(zero, v)
		}
	}
	sort.Strings(zero)

	order := make([]string, 0, len(indeg))

	for len(zero) > 0 {
		// Pop first (deterministic)
		v := zero[0]
		zero = zero[1:]
		order = append(order, v)

		// For each successor, decrement indegree
		for w := range succCopy[v] {
			indeg[w]--
			if indeg[w] == 0 {
				// Insert maintaining sorted order (N is small; simple append+sort)
				zero = append(zero, w)
			}
		}
		sort.Strings(zero)
	}

	if len(order) != len(indeg) {
		return nil, errors.New("dependency graph contains a cycle")
	}
	return order, nil
}

// HasCycles checks if the dependency graph contains cycles.
func (sdg *ServiceDependencyGraph) HasCycles() bool {
	order, err := sdg.GetTopologicalOrder()
	if err != nil {
		return true
	}

	sdg.mu.RLock()
	predLen := len(sdg.pred)
	sdg.mu.RUnlock()

	return len(order) != predLen
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
