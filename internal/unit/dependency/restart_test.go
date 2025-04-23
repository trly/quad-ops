package dependency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUnitName(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{
			name:     "project-service",
			expected: []string{"project", "service"},
		},
		{
			name:     "noprefix",
			expected: []string{"noprefix"},
		},
	}

	for _, tt := range tests {
		result := SplitUnitName(tt.name)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMarkDependentsAsRestarted(t *testing.T) {
	// Create a dependency tree
	dependencyTree := map[string]*ServiceDependency{
		"db": {
			Dependencies:      map[string]struct{}{},
			DependentServices: map[string]struct{}{"app": {}},
		},
		"app": {
			Dependencies:      map[string]struct{}{"db": {}},
			DependentServices: map[string]struct{}{"web": {}},
		},
		"web": {
			Dependencies:      map[string]struct{}{"app": {}},
			DependentServices: map[string]struct{}{},
		},
	}

	// Initialize the restarted map
	restarted := make(map[string]bool)

	// Mark db and its dependents as restarted
	MarkDependentsAsRestarted("db", dependencyTree, "test", restarted)

	// db, app, and web should all be marked as restarted
	assert.True(t, restarted["test-db.container"])
	assert.True(t, restarted["test-app.container"])
	assert.True(t, restarted["test-web.container"])
}

func TestIsServiceAlreadyRestarted(t *testing.T) {
	// Create a dependency tree
	dependencyTree := map[string]*ServiceDependency{
		"db": {
			Dependencies:      map[string]struct{}{},
			DependentServices: map[string]struct{}{"app": {}},
		},
		"app": {
			Dependencies:      map[string]struct{}{"db": {}},
			DependentServices: map[string]struct{}{"web": {}},
		},
		"web": {
			Dependencies:      map[string]struct{}{"app": {}},
			DependentServices: map[string]struct{}{},
		},
	}

	// Test cases
	tests := []struct {
		name      string
		restarted map[string]bool
		service   string
		expected  bool
	}{
		{
			name:      "direct service restarted",
			restarted: map[string]bool{"test-app.container": true},
			service:   "app",
			expected:  true,
		},
		{
			name:      "dependency restarted",
			restarted: map[string]bool{"test-db.container": true},
			service:   "app",
			expected:  true,
		},
		{
			name:      "dependent service restarted",
			restarted: map[string]bool{"test-web.container": true},
			service:   "app",
			expected:  false,
		},
		{
			name:      "unrelated service restarted",
			restarted: map[string]bool{"test-other.container": true},
			service:   "app",
			expected:  false,
		},
		{
			name:      "no services restarted",
			restarted: map[string]bool{},
			service:   "app",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServiceAlreadyRestarted(tt.service, dependencyTree, "test", tt.restarted)
			assert.Equal(t, tt.expected, result)
		})
	}
}
