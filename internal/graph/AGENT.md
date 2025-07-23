# Agent Guidelines for dependency Package

**Parent docs**: [internal/AGENT.md](../AGENT.md) • [/AGENT.md](../../AGENT.md)

## Overview
The `dependency` package provides service dependency graph management for Docker Compose projects. It uses directed acyclic graphs to model service dependencies and enables topological ordering for proper startup/shutdown sequences.

## Key Structures and Functions

### Core Structures
- **`ServiceDependencyGraph`** - Main structure wrapping a directed graph:
  - `graph` - Internal directed acyclic graph implementation

### Key Functions
- **`NewServiceDependencyGraph()`** - Creates a new dependency graph
- **`AddService(serviceName string)`** - Adds a service vertex to the graph
- **`AddDependency(dependent, dependency string)`** - Adds dependency edge (dependency → dependent)
- **`GetDependencies(serviceName string)`** - Returns services this service depends on
- **`GetDependents(serviceName string)`** - Returns services that depend on this service
- **`GetTopologicalOrder()`** - Returns services in dependency order
- **`HasCycles()`** - Checks for circular dependencies
- **`BuildServiceDependencyGraph(project *types.Project)`** - Builds graph from Compose project

### Key Dependencies
- **`github.com/compose-spec/compose-go/v2/types`** - Docker Compose types
- **`github.com/dominikbraun/graph`** - Graph data structure library

## Usage Patterns

### Building a Dependency Graph
```go
// From a Docker Compose project
graph, err := dependency.BuildServiceDependencyGraph(project)
if err != nil {
    return fmt.Errorf("failed to build dependency graph: %w", err)
}

// Manual construction
graph := dependency.NewServiceDependencyGraph()
graph.AddService("web")
graph.AddService("db")
graph.AddDependency("web", "db") // web depends on db
```

### Analyzing Dependencies
```go
// Get what a service depends on
deps, err := graph.GetDependencies("web")
// deps = ["db"]

// Get what depends on a service
dependents, err := graph.GetDependents("db")
// dependents = ["web"]

// Get proper startup order
order, err := graph.GetTopologicalOrder()
// order = ["db", "web"] (dependencies first)
```

## Graph Properties
- **Directed**: Dependencies have direction (A depends on B, not necessarily vice versa)
- **Acyclic**: Circular dependencies are prevented at edge addition time
- **String vertices**: Service names are used as vertex identifiers
- **No self-loops**: Services cannot depend on themselves

## Dependency Direction
- Edge direction: `dependency → dependent`
- `AddDependency(web, db)` means "web depends on db"
- Topological order returns dependencies first

## Error Handling
- Cycle detection prevents invalid graphs
- Missing services in dependency relationships cause errors
- Graph operations return meaningful error messages

## Memory Efficiency
- Uses map-based adjacency representation
- Efficient for sparse graphs (typical in microservices)
- Vertices are strings (service names), not complex objects

## Common Patterns

### Safe Dependency Analysis
```go
// Always check for cycles before proceeding
if graph.HasCycles() {
    return fmt.Errorf("dependency graph contains cycles")
}

// Get safe startup order
order, err := graph.GetTopologicalOrder()
if err != nil {
    return fmt.Errorf("failed to compute startup order: %w", err)
}
```

## Graph Algorithm Notes

### Topological Sort
- Returns dependencies before dependents
- Multiple valid orderings may exist
- Empty slice returned for empty graph
- Error returned if cycles detected

### Cycle Detection
- Prevented at edge addition time (acyclic constraint)
- Can still be checked explicitly via `HasCycles()`
- Based on topological sort failure
