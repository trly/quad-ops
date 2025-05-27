package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/dependency"
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

func TestIsServiceAlreadyRestarted(t *testing.T) {
	// Create a dependency graph:
	// A <- B <- C
	dependencyGraph := dependency.NewServiceDependencyGraph()
	_ = dependencyGraph.AddService("A")
	_ = dependencyGraph.AddService("B")
	_ = dependencyGraph.AddService("C")
	_ = dependencyGraph.AddDependency("B", "A")
	_ = dependencyGraph.AddDependency("C", "B")

	// Test 1: No services restarted yet
	restarted := make(map[string]bool)
	assert.False(t, isServiceAlreadyRestarted("A", dependencyGraph, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("B", dependencyGraph, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("C", dependencyGraph, "project", restarted))

	// Test 2: Restart C, check if B or A consider themselves already restarted
	// (they shouldn't since their dependencies aren't restarted)
	restarted["project-C.container"] = true
	assert.False(t, isServiceAlreadyRestarted("A", dependencyGraph, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("B", dependencyGraph, "project", restarted))
	assert.True(t, isServiceAlreadyRestarted("C", dependencyGraph, "project", restarted))

	// Test 3: Restart A, check if its dependent services (B) consider themselves already restarted
	restarted = make(map[string]bool)
	restarted["project-A.container"] = true
	assert.True(t, isServiceAlreadyRestarted("A", dependencyGraph, "project", restarted))
	assert.True(t, isServiceAlreadyRestarted("B", dependencyGraph, "project", restarted))
	assert.False(t, isServiceAlreadyRestarted("C", dependencyGraph, "project", restarted))
}
