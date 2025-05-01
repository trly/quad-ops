package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUnitName(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{"project-service", []string{"project", "service"}},
		{"multi-part-project-service", []string{"multi-part-project", "service"}},
		{"single", []string{"single"}},
		{"", []string{""}},
	}

	for _, tt := range tests {
		result := splitUnitName(tt.name)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFindTopLevelDependentService(t *testing.T) {
	// Create a dependency tree:
	// A <- B <- C
	// |    ^
	// v    |
	// D <- E
	dependencyTree := map[string]*ServiceDependency{
		"A": {
			Dependencies:      make(map[string]struct{}),
			DependentServices: map[string]struct{}{"B": {}, "D": {}},
		},
		"B": {
			Dependencies:      map[string]struct{}{"A": {}},
			DependentServices: map[string]struct{}{"C": {}},
		},
		"C": {
			Dependencies:      map[string]struct{}{"B": {}},
			DependentServices: make(map[string]struct{}),
		},
		"D": {
			Dependencies:      map[string]struct{}{"A": {}},
			DependentServices: map[string]struct{}{"E": {}},
		},
		"E": {
			Dependencies:      map[string]struct{}{"D": {}},
			DependentServices: map[string]struct{}{"B": {}},
		},
	}

	// Test finding top-level dependent service
	tests := []struct {
		serviceName string
		expected    string
	}{
		{"A", "C"}, // A -> B -> C (C is top-level)
		{"B", "C"}, // B -> C (C is top-level)
		{"C", ""},  // C has no dependents
		{"D", "C"}, // D -> E -> B -> C (C is top-level)
		{"E", "C"}, // E -> B -> C (C is top-level)
	}

	for _, tt := range tests {
		result := findTopLevelDependentService(tt.serviceName, dependencyTree)
		assert.Equal(t, tt.expected, result, "For service %s", tt.serviceName)
	}
}

func TestMarkDependentsAsRestarted(t *testing.T) {
	// Create a dependency tree:
	// A <- B <- C
	dependencyTree := map[string]*ServiceDependency{
		"A": {
			Dependencies:      make(map[string]struct{}),
			DependentServices: map[string]struct{}{"B": {}},
		},
		"B": {
			Dependencies:      map[string]struct{}{"A": {}},
			DependentServices: map[string]struct{}{"C": {}},
		},
		"C": {
			Dependencies:      map[string]struct{}{"B": {}},
			DependentServices: make(map[string]struct{}),
		},
	}

	// Test marking dependents as restarted
	restarted := make(map[string]bool)
	markDependentsAsRestarted("A", dependencyTree, "project", restarted)

	// Should mark A, B, and C as restarted
	assert.True(t, restarted["project-A.container"])
	assert.True(t, restarted["project-B.container"])
	assert.True(t, restarted["project-C.container"])

	// Try starting from B
	restarted = make(map[string]bool)
	markDependentsAsRestarted("B", dependencyTree, "project", restarted)

	// Should mark B and C as restarted, but not A
	assert.False(t, restarted["project-A.container"])
	assert.True(t, restarted["project-B.container"])
	assert.True(t, restarted["project-C.container"])
}

func TestIsServiceAlreadyRestarted(t *testing.T) {
	// Create a dependency tree:
	// A <- B <- C
	dependencyTree := map[string]*ServiceDependency{
		"A": {
			Dependencies:      make(map[string]struct{}),
			DependentServices: map[string]struct{}{"B": {}},
		},
		"B": {
			Dependencies:      map[string]struct{}{"A": {}},
			DependentServices: map[string]struct{}{"C": {}},
		},
		"C": {
			Dependencies:      map[string]struct{}{"B": {}},
			DependentServices: make(map[string]struct{}),
		},
	}

	// Test 1: No services restarted yet
	restarted := make(map[string]bool)
	assert.False(t, isServiceAlreadyRestarted("A", dependencyTree, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("B", dependencyTree, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("C", dependencyTree, "project", restarted))

	// Test 2: Restart C, check if B or A consider themselves already restarted
	// (they shouldn't since their dependencies aren't restarted)
	restarted["project-C.container"] = true
	assert.False(t, isServiceAlreadyRestarted("A", dependencyTree, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("B", dependencyTree, "project", restarted))
	assert.True(t, isServiceAlreadyRestarted("C", dependencyTree, "project", restarted))

	// Test 3: Restart A, check if its dependent services (B) consider themselves already restarted
	restarted = make(map[string]bool)
	restarted["project-A.container"] = true
	assert.True(t, isServiceAlreadyRestarted("A", dependencyTree, "project", restarted))
	assert.True(t, isServiceAlreadyRestarted("B", dependencyTree, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("C", dependencyTree, "project", restarted))
}
