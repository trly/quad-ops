package dependency

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/google/go-cmp/cmp"
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

	// Check topological order works
	order, err := graph.GetTopologicalOrder()
	require.NoError(t, err)
	want := []string{"a", "b", "c"}
	if diff := cmp.Diff(want, order); diff != "" {
		t.Errorf("topological order mismatch (-want +got):\n%s", diff)
	}

	// Try to add c -> a, which would create a cycle: a -> b -> c -> a
	err = graph.AddDependency("a", "c")
	require.NoError(t, err) // AddDependency doesn't prevent cycles

	// Now should have cycles
	assert.True(t, graph.HasCycles())

	// Check topological order fails with improved error message
	_, err = graph.GetTopologicalOrder()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency graph contains a cycle involving services:")
	assert.Contains(t, err.Error(), "a")
	assert.Contains(t, err.Error(), "b")
	assert.Contains(t, err.Error(), "c")
}

func TestGetTransitiveDependencies(t *testing.T) {
	// Create a mock project with a dependency tree
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

	// Test transitive dependencies
	deps, err := graph.GetTransitiveDependencies("db")
	require.NoError(t, err)
	assert.Empty(t, deps)

	deps, err = graph.GetTransitiveDependencies("webapp")
	require.NoError(t, err)
	want := []string{"db"}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("webapp transitive dependencies mismatch (-want +got):\n%s", diff)
	}

	deps, err = graph.GetTransitiveDependencies("proxy")
	require.NoError(t, err)
	want = []string{"db", "webapp"}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("proxy transitive dependencies mismatch (-want +got):\n%s", diff)
	}

	// Test unknown service
	_, err = graph.GetTransitiveDependencies("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}

func TestGetTransitiveDependents(t *testing.T) {
	// Create a mock project with a dependency tree
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

	// Test transitive dependents
	deps, err := graph.GetTransitiveDependents("db")
	require.NoError(t, err)
	want := []string{"proxy", "webapp"}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("db transitive dependents mismatch (-want +got):\n%s", diff)
	}

	deps, err = graph.GetTransitiveDependents("webapp")
	require.NoError(t, err)
	want = []string{"proxy"}
	if diff := cmp.Diff(want, deps); diff != "" {
		t.Errorf("webapp transitive dependents mismatch (-want +got):\n%s", diff)
	}

	deps, err = graph.GetTransitiveDependents("proxy")
	require.NoError(t, err)
	assert.Empty(t, deps)

	// Test unknown service
	_, err = graph.GetTransitiveDependents("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}

func TestCanAddDependency(t *testing.T) {
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

	// Can add existing dependency
	can, err := graph.CanAddDependency("b", "a")
	require.NoError(t, err)
	assert.True(t, can)

	// Can add new valid dependency
	can, err = graph.CanAddDependency("c", "a")
	require.NoError(t, err)
	assert.True(t, can)

	// Cannot add self-dependency
	_, err = graph.CanAddDependency("a", "a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "self-dependency is not allowed")

	// Cannot add cycle
	can, err = graph.CanAddDependency("a", "c")
	require.NoError(t, err)
	assert.False(t, can)

	// Unknown service
	_, err = graph.CanAddDependency("unknown", "a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dependent service")

	_, err = graph.CanAddDependency("a", "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dependency service")
}
