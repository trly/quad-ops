package unit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddBuildBasicConfig tests the addBuildBasicConfig refactored method
func TestAddBuildBasicConfig(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			ImageTag:            []string{"test:latest", "test:v1.0"},
			File:                "Dockerfile.prod",
			SetWorkingDirectory: "/app/src",
		},
	}

	var builder strings.Builder
	quadletUnit.addBuildBasicConfig(&builder)
	result := builder.String()

	// Verify image tags are included
	assert.Contains(t, result, "ImageTag=test:latest")
	assert.Contains(t, result, "ImageTag=test:v1.0")
	assert.Contains(t, result, "File=Dockerfile.prod")
	assert.Contains(t, result, "SetWorkingDirectory=/app/src")
}

// TestAddBuildMetadata tests the addBuildMetadata refactored method
func TestAddBuildMetadata(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			Label:      []string{"version=1.0", "team=backend"},
			Annotation: []string{"description=Test build", "maintainer=dev-team"},
		},
	}

	var builder strings.Builder
	quadletUnit.addBuildMetadata(&builder)
	result := builder.String()

	// Verify labels and annotations are included
	assert.Contains(t, result, "Label=team=backend")
	assert.Contains(t, result, "Label=version=1.0")
	assert.Contains(t, result, "Annotation=description=Test build")
	assert.Contains(t, result, "Annotation=maintainer=dev-team")
}

// TestAddBuildEnvironment tests the addBuildEnvironment refactored method
func TestAddBuildEnvironment(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			Env: map[string]string{
				"NODE_ENV":    "production",
				"API_VERSION": "v2",
				"DEBUG":       "false",
			},
		},
	}

	var builder strings.Builder
	quadletUnit.addBuildEnvironment(&builder)
	result := builder.String()

	// Verify environment variables are included and sorted
	assert.Contains(t, result, "Environment=API_VERSION=v2")
	assert.Contains(t, result, "Environment=DEBUG=false")
	assert.Contains(t, result, "Environment=NODE_ENV=production")
	
	// Verify they appear in sorted order
	apiIndex := strings.Index(result, "Environment=API_VERSION=v2")
	debugIndex := strings.Index(result, "Environment=DEBUG=false")
	nodeIndex := strings.Index(result, "Environment=NODE_ENV=production")
	
	assert.True(t, apiIndex < debugIndex, "API_VERSION should come before DEBUG")
	assert.True(t, debugIndex < nodeIndex, "DEBUG should come before NODE_ENV")
}

// TestAddBuildResources tests the addBuildResources refactored method
func TestAddBuildResources(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			Network: []string{"host", "build-network"},
			Volume:  []string{"/tmp:/tmp", "cache-vol:/cache"},
			Secret:  []string{"api-key", "db-password"},
		},
	}

	var builder strings.Builder
	quadletUnit.addBuildResources(&builder)
	result := builder.String()

	// Verify resources are included
	assert.Contains(t, result, "Network=build-network")
	assert.Contains(t, result, "Network=host")
	assert.Contains(t, result, "Volume=/tmp:/tmp")
	assert.Contains(t, result, "Volume=cache-vol:/cache")
	assert.Contains(t, result, "Secret=api-key")
	assert.Contains(t, result, "Secret=db-password")
}

// TestAddBuildOptions tests the addBuildOptions refactored method
func TestAddBuildOptions(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			Target:     "production",
			Pull:       "always",
			PodmanArgs: []string{"--no-cache", "--squash"},
		},
	}

	var builder strings.Builder
	quadletUnit.addBuildOptions(&builder)
	result := builder.String()

	// Verify build options are included
	assert.Contains(t, result, "Target=production")
	assert.Contains(t, result, "Pull=always")
	assert.Contains(t, result, "PodmanArgs=--no-cache")
	assert.Contains(t, result, "PodmanArgs=--squash")
}

// TestGenerateBuildSectionIntegration tests the full generateBuildSection method
func TestGenerateBuildSectionIntegration(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type: "build",
		Build: Build{
			ImageTag:            []string{"test:latest"},
			File:                "Dockerfile",
			SetWorkingDirectory: "/app",
			Label:               []string{"version=1.0"},
			Annotation:          []string{"description=Test"},
			Env: map[string]string{
				"NODE_ENV": "production",
			},
			Network:    []string{"host"},
			Volume:     []string{"/tmp:/tmp"},
			Secret:     []string{"api-key"},
			Target:     "prod",
			Pull:       "always",
			PodmanArgs: []string{"--no-cache"},
		},
	}

	result := quadletUnit.generateBuildSection()

	// Verify section header and managed-by label
	assert.Contains(t, result, "[Build]")
	assert.Contains(t, result, "Label=managed-by=quad-ops")

	// Verify all components are present
	assert.Contains(t, result, "ImageTag=test:latest")
	assert.Contains(t, result, "File=Dockerfile")
	assert.Contains(t, result, "SetWorkingDirectory=/app")
	assert.Contains(t, result, "Label=version=1.0")
	assert.Contains(t, result, "Annotation=description=Test")
	assert.Contains(t, result, "Environment=NODE_ENV=production")
	assert.Contains(t, result, "Network=host")
	assert.Contains(t, result, "Volume=/tmp:/tmp")
	assert.Contains(t, result, "Secret=api-key")
	assert.Contains(t, result, "Target=prod")
	assert.Contains(t, result, "Pull=always")
	assert.Contains(t, result, "PodmanArgs=--no-cache")
}

// TestEmptyBuildSection tests generateBuildSection with minimal configuration
func TestEmptyBuildSection(t *testing.T) {
	quadletUnit := &QuadletUnit{
		Type:  "build",
		Build: Build{}, // Empty build config
	}

	result := quadletUnit.generateBuildSection()

	// Should still have section header and managed-by label
	assert.Contains(t, result, "[Build]")
	assert.Contains(t, result, "Label=managed-by=quad-ops")
	
	// Should not contain any empty values
	assert.NotContains(t, result, "ImageTag=")
	assert.NotContains(t, result, "File=")
	assert.NotContains(t, result, "SetWorkingDirectory=")
}