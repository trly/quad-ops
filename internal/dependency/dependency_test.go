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
	assert.Contains(t, err.Error(), "dependency graph contains a cycle:")
	assert.Contains(t, err.Error(), "→", "error should show cycle path with arrows")
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

func TestFindCycle(t *testing.T) {
	tests := []struct {
		name         string
		setupGraph   func(*ServiceDependencyGraph)
		expectCycle  bool
		expectedPath []string
		pathContains []string // Services that must be in the cycle path
	}{
		{
			name: "simple cycle A→B→A",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
				_ = g.AddService("b")
				_ = g.AddDependency("b", "a") // b depends on a
				_ = g.AddDependency("a", "b") // a depends on b (creates cycle)
			},
			expectCycle:  true,
			pathContains: []string{"a", "b"},
		},
		{
			name: "complex cycle A→B→C→D→B",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
				_ = g.AddService("b")
				_ = g.AddService("c")
				_ = g.AddService("d")
				_ = g.AddDependency("b", "a") // b depends on a
				_ = g.AddDependency("c", "b") // c depends on b
				_ = g.AddDependency("d", "c") // d depends on c
				_ = g.AddDependency("b", "d") // b depends on d (creates cycle b→c→d→b)
			},
			expectCycle:  true,
			pathContains: []string{"b", "c", "d"},
		},
		{
			name: "three node cycle web→api→db→web",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("web")
				_ = g.AddService("api")
				_ = g.AddService("db")
				_ = g.AddDependency("api", "web") // api depends on web
				_ = g.AddDependency("db", "api")  // db depends on api
				_ = g.AddDependency("web", "db")  // web depends on db (creates cycle)
			},
			expectCycle:  true,
			pathContains: []string{"web", "api", "db"},
		},
		{
			name: "no cycle - linear chain",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
				_ = g.AddService("b")
				_ = g.AddService("c")
				_ = g.AddDependency("b", "a") // b depends on a
				_ = g.AddDependency("c", "b") // c depends on b
			},
			expectCycle: false,
		},
		{
			name: "no cycle - diamond dependency",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
				_ = g.AddService("b")
				_ = g.AddService("c")
				_ = g.AddService("d")
				_ = g.AddDependency("b", "a") // b depends on a
				_ = g.AddDependency("c", "a") // c depends on a
				_ = g.AddDependency("d", "b") // d depends on b
				_ = g.AddDependency("d", "c") // d depends on c
			},
			expectCycle: false,
		},
		{
			name: "no cycle - empty graph",
			setupGraph: func(_ *ServiceDependencyGraph) {
				// No services added
			},
			expectCycle: false,
		},
		{
			name: "no cycle - single service",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
			},
			expectCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewServiceDependencyGraph()
			tt.setupGraph(graph)

			cycle, err := graph.FindCycle()
			require.NoError(t, err)

			if tt.expectCycle {
				assert.NotEmpty(t, cycle, "expected to find a cycle")
				// Verify all expected services are in the cycle path
				for _, svc := range tt.pathContains {
					assert.Contains(t, cycle, svc, "cycle path should contain %s", svc)
				}
				// Cycle path should form a loop (first and last should be same)
				if len(cycle) > 1 {
					assert.Equal(t, cycle[0], cycle[len(cycle)-1], "cycle path should start and end with same service")
				}
			} else {
				assert.Empty(t, cycle, "expected no cycle")
			}
		})
	}
}

func TestGetTopologicalOrderWithEnhancedCycleError(t *testing.T) {
	tests := []struct {
		name              string
		setupGraph        func(*ServiceDependencyGraph)
		expectError       bool
		errorContains     []string
		cyclePathContains []string
	}{
		{
			name: "simple cycle with enhanced error",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("web")
				_ = g.AddService("api")
				_ = g.AddDependency("api", "web")
				_ = g.AddDependency("web", "api")
			},
			expectError: true,
			errorContains: []string{
				"dependency graph contains a cycle",
				"web",
				"api",
			},
			cyclePathContains: []string{"web", "api"},
		},
		{
			name: "complex cycle with enhanced error",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("web")
				_ = g.AddService("api")
				_ = g.AddService("db")
				_ = g.AddDependency("api", "web")
				_ = g.AddDependency("db", "api")
				_ = g.AddDependency("web", "db")
			},
			expectError: true,
			errorContains: []string{
				"dependency graph contains a cycle",
				"web",
				"api",
				"db",
			},
			cyclePathContains: []string{"web", "api", "db"},
		},
		{
			name: "no cycle returns success",
			setupGraph: func(g *ServiceDependencyGraph) {
				_ = g.AddService("a")
				_ = g.AddService("b")
				_ = g.AddService("c")
				_ = g.AddDependency("b", "a")
				_ = g.AddDependency("c", "b")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewServiceDependencyGraph()
			tt.setupGraph(graph)

			order, err := graph.GetTopologicalOrder()

			if tt.expectError {
				require.Error(t, err)
				for _, substr := range tt.errorContains {
					assert.Contains(t, err.Error(), substr)
				}
				// Verify error message is actionable
				assert.Contains(t, err.Error(), "→", "error should show cycle path with arrows")
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, order)
			}
		})
	}
}
