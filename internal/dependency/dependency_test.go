package dependency

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesDependencyGraph(t *testing.T) {
	// Create a mock project with a simple dependency tree
	// db <- webapp <- proxy
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"db": types.ServiceConfig{
				Name:  "db",
				Image: "mariadb:latest",
			},
			"webapp": types.ServiceConfig{
				Name:  "webapp",
				Image: "wordpress:latest",
				DependsOn: types.DependsOnConfig{
					"db": types.ServiceDependency{},
				},
			},
			"proxy": types.ServiceConfig{
				Name:  "proxy",
				Image: "nginx:latest",
				DependsOn: types.DependsOnConfig{
					"webapp": types.ServiceDependency{},
				},
			},
		},
	}

	// Build the dependency graph
	graph, err := BuildServiceDependencyGraph(project)
	require.NoError(t, err)

	// Check db has no dependencies
	deps, err := graph.GetDependencies("db")
	require.NoError(t, err)
	assert.Empty(t, deps)

	dependents, err := graph.GetDependents("db")
	require.NoError(t, err)
	assert.Len(t, dependents, 1)
	assert.Contains(t, dependents, "webapp")

	// Check webapp has db as dependency
	deps, err = graph.GetDependencies("webapp")
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "db")

	dependents, err = graph.GetDependents("webapp")
	require.NoError(t, err)
	assert.Len(t, dependents, 1)
	assert.Contains(t, dependents, "proxy")

	// Check proxy has webapp as dependency
	deps, err = graph.GetDependencies("proxy")
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "webapp")

	dependents, err = graph.GetDependents("proxy")
	require.NoError(t, err)
	assert.Empty(t, dependents)

	// Check topological order
	order, err := graph.GetTopologicalOrder()
	require.NoError(t, err)
	assert.Len(t, order, 3)

	// db should come before webapp, webapp should come before proxy
	dbIndex := -1
	webappIndex := -1
	proxyIndex := -1
	for i, service := range order {
		switch service {
		case "db":
			dbIndex = i
		case "webapp":
			webappIndex = i
		case "proxy":
			proxyIndex = i
		}
	}

	assert.True(t, dbIndex < webappIndex, "db should come before webapp")
	assert.True(t, webappIndex < proxyIndex, "webapp should come before proxy")

	// Check that there are no cycles
	assert.False(t, graph.HasCycles())
}

func TestDependencyGraphWithBuildDependency(t *testing.T) {
	// Create a simple project with one service that has a build
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"webapp": types.ServiceConfig{
				Name:  "webapp",
				Image: "webapp:latest",
				Build: &types.BuildConfig{
					Context: ".",
				},
			},
		},
	}

	// Build the dependency graph
	graph, err := BuildServiceDependencyGraph(project)
	require.NoError(t, err)

	// Initially, webapp should have no dependencies
	deps, err := graph.GetDependencies("webapp")
	require.NoError(t, err)
	assert.Empty(t, deps)

	// Simulate adding a build dependency (this would be done by the build processor)
	err = graph.AddService("webapp-build")
	require.NoError(t, err)
	err = graph.AddDependency("webapp", "webapp-build")
	require.NoError(t, err)

	// Now webapp should have webapp-build as a dependency
	deps, err = graph.GetDependencies("webapp")
	require.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "webapp-build")

	// webapp-build should have webapp as dependent
	dependents, err := graph.GetDependents("webapp-build")
	require.NoError(t, err)
	assert.Len(t, dependents, 1)
	assert.Contains(t, dependents, "webapp")
}

func TestDependencyGraphCycleDetection(t *testing.T) {
	// Test that cycles are detected
	graph := NewServiceDependencyGraph()

	// Add services
	err := graph.AddService("a")
	require.NoError(t, err)
	err = graph.AddService("b")
	require.NoError(t, err)
	err = graph.AddService("c")
	require.NoError(t, err)

	// Add valid dependencies: a -> b -> c
	err = graph.AddDependency("b", "a")
	require.NoError(t, err)
	err = graph.AddDependency("c", "b")
	require.NoError(t, err)

	// No cycles yet
	assert.False(t, graph.HasCycles())

	// Try to add c -> a, which would create a cycle: a -> b -> c -> a
	// The graph uses graph.Acyclic() which should prevent this
	err = graph.AddDependency("a", "c")
	if err == nil {
		// If the edge was added successfully, the HasCycles should detect it
		assert.True(t, graph.HasCycles(), "Cycle should be detected after adding cyclic dependency")
	} else {
		// If the edge addition failed (which is expected with Acyclic), that's also fine
		assert.Error(t, err, "Graph correctly rejected cyclic dependency")
	}
}
