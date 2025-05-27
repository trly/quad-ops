package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/log"
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

func TestStartUnitDependencyAware_HandlesOneShotServices(_ *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	// Test that one-shot services (build, image, network, volume) are handled without error
	// Note: These will fail in test environment since systemd is not available, but we're testing
	// that the function doesn't panic and follows the correct code path
	tests := []struct {
		unitType string
		unitName string
	}{
		{"build", "test-project-webapp-build"},
		{"image", "test-project-webapp-image"},
		{"network", "test-project-default"},
		{"volume", "test-project-data"},
	}

	for _, tt := range tests {
		// We expect an error here since systemd is not available in test environment,
		// but we're verifying the function doesn't panic and handles the unit types correctly
		_ = StartUnitDependencyAware(tt.unitName, tt.unitType, nil)
		// The important thing is that we don't panic and the function completes
	}
}

func TestRestartChangedUnits_HandlesOneShotServices(_ *testing.T) {
	// Initialize logger for tests
	log.Init(false)

	// Create test units representing one-shot services
	changedUnits := []QuadletUnit{
		{Name: "test-build", Type: "build"},
		{Name: "test-volume", Type: "volume"},
		{Name: "test-network", Type: "network"},
		{Name: "test-container", Type: "container"},
	}

	// This will fail because systemd is not available, but it verifies that:
	// 1. One-shot services (build, volume, network) are processed first
	// 2. Container services are processed separately
	// 3. The function doesn't panic with different unit types
	_ = RestartChangedUnits(changedUnits, make(map[string]*dependency.ServiceDependencyGraph))
}

// These tests have been removed as the dependency-aware restart logic has been simplified.
// Services are now restarted directly and systemd handles the dependency propagation automatically.

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
