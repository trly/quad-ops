package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestNetworkAliasGeneration(t *testing.T) {
	// Create a mock container service
	cfg := &config.Config{
		UsePodmanDefaultNames: false,
	}
	config.SetConfig(cfg)

	// Create a service with a simple name
	serviceConfig := types.ServiceConfig{
		Name:  "db",
		Image: "mariadb:latest",
	}

	// Create a container from this service
	prefixedName := "test-project-db"
	container := NewContainer(prefixedName)
	container = container.FromComposeService(serviceConfig, "test-project")

	// Set ContainerName to simulate what compose_processor does when UsePodmanDefaultNames is false
	container.ContainerName = prefixedName

	// Now add the service name as a NetworkAlias (simulating what compose_processor does)
	container.NetworkAlias = append(container.NetworkAlias, "db")

	// Verify the service name is added as a NetworkAlias
	assert.Contains(t, container.NetworkAlias, "db")

	// Generate the quadlet unit content to verify it includes the NetworkAlias directive
	quadletUnit := QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
		Systemd:   SystemdConfig{},
	}

	// Generate the unit content
	content := GenerateQuadletUnit(quadletUnit, false)

	// Verify the NetworkAlias directive is included in the output
	assert.Contains(t, content, "NetworkAlias=db")

	// Test with a container name (which should be set because UsePodmanDefaultNames is false)
	assert.Contains(t, content, "ContainerName=test-project-db")

	// Change the config to use Podman default names
	cfg.UsePodmanDefaultNames = true
	config.SetConfig(cfg)

	// Create a new container with the updated config
	container2 := NewContainer(prefixedName)
	container2 = container2.FromComposeService(serviceConfig, "test-project")
	container2.NetworkAlias = append(container2.NetworkAlias, "db")

	// Create a new quadlet unit
	quadletUnit2 := QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container2,
		Systemd:   SystemdConfig{},
	}

	// Generate the unit content
	content2 := GenerateQuadletUnit(quadletUnit2, false)

	// Verify the NetworkAlias directive is still included
	assert.Contains(t, content2, "NetworkAlias=db")

	// But ContainerName should NOT be set when UsePodmanDefaultNames is true
	assert.NotContains(t, content2, "ContainerName=")

	// Test with existing network aliases in a compose file
	// Create a service with network aliases
	serviceWithAliases := types.ServiceConfig{
		Name:  "web",
		Image: "nginx:latest",
		Networks: map[string]*types.ServiceNetworkConfig{
			"frontend": {
				Aliases: []string{"www", "website"},
			},
		},
	}

	// Create a container from this service
	prefixedName3 := "test-project-web"
	container3 := NewContainer(prefixedName3)
	container3 = container3.FromComposeService(serviceWithAliases, "test-project")

	// Manually add the service name as a NetworkAlias (as the processor would do)
	container3.NetworkAlias = append(container3.NetworkAlias, "web")

	// Verify all network aliases are present: both from the compose file and the added service name
	assert.Contains(t, container3.NetworkAlias, "web")
	assert.Contains(t, container3.NetworkAlias, "www")
	assert.Contains(t, container3.NetworkAlias, "website")

	// Reset config to default for other tests
	cfg.UsePodmanDefaultNames = false
	config.SetConfig(cfg)
}
