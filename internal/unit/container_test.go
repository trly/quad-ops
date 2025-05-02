package unit

import (
	"crypto/sha1" //nolint:gosec // Not used for security purposes, just content comparison
	"fmt"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestDeterministicUnitContent(t *testing.T) {
	// Create two identical container configs
	container1 := NewContainer("test-container")
	container2 := NewContainer("test-container")

	// Configure them identically
	service := types.ServiceConfig{
		Image: "docker.io/test:latest",
		Environment: types.MappingWithEquals{
			"VAR1": &[]string{"value1"}[0],
			"VAR2": &[]string{"value2"}[0],
		},
	}

	// Convert to container objects
	container1.FromComposeService(service, "test-project")
	// Do it again for the second container
	container2.FromComposeService(service, "test-project")

	// Generate quadlet units
	unit1 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container1,
	}

	unit2 := &QuadletUnit{
		Name:      "test-container",
		Type:      "container",
		Container: *container2,
	}

	// Generate the content - this should be deterministic
	content1 := GenerateQuadletUnit(*unit1, false)
	content2 := GenerateQuadletUnit(*unit2, false)

	// Should produce identical content
	assert.Equal(t, content1, content2, "Unit content should be identical when generated from identical configs")

	// Check if content hashing is deterministic
	hash1 := GetContentHash(content1)
	hash2 := GetContentHash(content2)

	assert.Equal(t, hash1, hash2, "Content hashes should be identical for identical content")
}

func GetContentHash(content string) string {
	hash := sha1.New() //nolint:gosec // Not used for security purposes
	hash.Write([]byte(content))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func TestCustomHostnameNetworkAlias(t *testing.T) {
	// Create mock configuration
	cfg := config.InitConfig() // Initialize default config
	config.SetConfig(cfg)

	// Create basic service with custom hostname
	service := types.ServiceConfig{
		Name:     "db",
		Image:    "docker.io/postgres:latest",
		Hostname: "photoprism-db", // Custom hostname
	}

	// Create project
	project := types.Project{
		Name: "test-project",
		Services: types.Services{
			"db": service,
		},
	}

	// Process the container
	prefixedName := fmt.Sprintf("%s-%s", project.Name, "db")
	container := NewContainer(prefixedName)
	container = container.FromComposeService(service, project.Name)

	// Set custom container name (usePodmanNames=false)
	container.ContainerName = prefixedName

	// Add service name as network alias
	container.NetworkAlias = append(container.NetworkAlias, "db")

	// Add custom hostname as network alias if specified
	if container.HostName != "" && container.HostName != "db" {
		container.NetworkAlias = append(container.NetworkAlias, container.HostName)
	}

	// Check that both the service name and custom hostname are in network aliases
	assert.Contains(t, container.NetworkAlias, "db", "Service name should be included in network aliases")
	assert.Contains(t, container.NetworkAlias, "photoprism-db", "Custom hostname should be included in network aliases")

	// Generate unit to verify output
	unit := &QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
	}

	// Generate the content and check it contains both network aliases
	content := GenerateQuadletUnit(*unit, false)
	assert.Contains(t, content, "NetworkAlias=db", "Unit content should include service name as NetworkAlias")
	assert.Contains(t, content, "NetworkAlias=photoprism-db", "Unit content should include custom hostname as NetworkAlias")
}
