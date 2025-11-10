package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

// TestNetworkDependencies_ServiceWithExplicitNetworks tests that a service
// explicitly declaring networks gets those networks in ServiceNetworks.
func TestNetworkDependencies_ServiceWithExplicitNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"frontend": {
				Name:   "frontend",
				Driver: "bridge",
			},
			"backend": {
				Name:   "backend",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"frontend": {},
					"backend":  {},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "myapp-web", spec.Name)

	// Verify the container has both networks in ServiceNetworks
	assert.ElementsMatch(t, []string{"myapp-backend", "myapp-frontend"}, spec.Container.Network.ServiceNetworks)

	// Verify spec.Networks contains both networks
	require.Len(t, spec.Networks, 2)
	networkNames := []string{spec.Networks[0].Name, spec.Networks[1].Name}
	assert.ElementsMatch(t, []string{"myapp-backend", "myapp-frontend"}, networkNames)
}

// TestNetworkDependencies_ServiceWithoutExplicitNetworks tests that a service
// WITHOUT explicit network declarations gets the project's default networks
// in ServiceNetworks.
func TestNetworkDependencies_ServiceWithoutExplicitNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:     "web",
				Image:    "nginx:latest",
				Networks: nil, // No explicit networks
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "myapp-web", spec.Name)

	// CRITICAL: The container should have the default network in ServiceNetworks
	// This is the bug we're fixing - currently this would be empty
	assert.Contains(t, spec.Container.Network.ServiceNetworks, "myapp-default",
		"Service without explicit networks should use project default networks in ServiceNetworks")

	// Verify spec.Networks contains the default network
	require.Len(t, spec.Networks, 1)
	assert.Equal(t, "myapp-default", spec.Networks[0].Name)
}

// TestNetworkDependencies_ServiceWithMultipleDefaultNetworks tests that a service
// without explicit networks gets ALL project default networks in ServiceNetworks.
func TestNetworkDependencies_ServiceWithMultipleDefaultNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
			"monitoring": {
				Name:   "monitoring",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:     "web",
				Image:    "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{}, // Empty but not nil
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "myapp-web", spec.Name)

	// Service with empty networks should get ALL project networks in ServiceNetworks
	assert.ElementsMatch(t, []string{"myapp-default", "myapp-monitoring"}, spec.Container.Network.ServiceNetworks,
		"Service with empty networks map should use all project networks in ServiceNetworks")

	// Verify spec.Networks contains all project networks
	require.Len(t, spec.Networks, 2)
	networkNames := []string{spec.Networks[0].Name, spec.Networks[1].Name}
	assert.ElementsMatch(t, []string{"myapp-default", "myapp-monitoring"}, networkNames)
}

// TestNetworkDependencies_ExternalNetworksNotInServiceNetworks tests that
// external networks are included in spec.Networks but also in ServiceNetworks
// for proper dependency tracking.
func TestNetworkDependencies_ExternalNetworksInServiceNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
			"infrastructure-proxy": {
				Name:     "infrastructure-proxy",
				External: types.External(true),
			},
		},
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"default":              {},
					"infrastructure-proxy": {},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// External networks should be in ServiceNetworks (for Network= directive)
	// but NOT have .network unit dependencies (handled by renderer)
	assert.ElementsMatch(t, []string{"infrastructure-proxy", "myapp-default"}, spec.Container.Network.ServiceNetworks)

	// spec.Networks should contain both networks
	require.Len(t, spec.Networks, 2)
	var externalNet *service.Network
	for i := range spec.Networks {
		if spec.Networks[i].Name == "infrastructure-proxy" {
			externalNet = &spec.Networks[i]
			break
		}
	}
	require.NotNil(t, externalNet)
	assert.True(t, externalNet.External, "infrastructure-proxy should be marked as external")
}

// TestNetworkDependencies_ExternalNetworkNotInProjectNetworks tests that
// external networks referenced by services but not defined in project.Networks
// are handled correctly without project prefix.
func TestNetworkDependencies_ExternalNetworkNotInProjectNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	// Simulate scenario where service references an external network
	// (from another project) that is NOT in the current project's Networks
	project := &types.Project{
		Name: "llm",
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name:   "default",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"ollama": {
				Name:  "ollama",
				Image: "ollama:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"default":              {}, // Local network
					"infrastructure-proxy": {}, // External network NOT in project.Networks
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// The external network should be in ServiceNetworks WITHOUT the project prefix
	// Expected: "infrastructure-proxy" (not "llm-infrastructure-proxy")
	assert.ElementsMatch(t,
		[]string{"infrastructure-proxy", "llm-default"},
		spec.Container.Network.ServiceNetworks,
		"external network should not have project prefix")

	// spec.Networks should have both networks
	require.Len(t, spec.Networks, 2)
	var externalNet *service.Network
	for i := range spec.Networks {
		if spec.Networks[i].Name == "infrastructure-proxy" {
			externalNet = &spec.Networks[i]
			break
		}
	}
	require.NotNil(t, externalNet, "infrastructure-proxy should exist in Networks")
	assert.True(t, externalNet.External, "external network should be marked as external")
}

// TestNetworkDependencies_BridgeMode tests that bridge mode services
// still get proper network assignments.
func TestNetworkDependencies_BridgeModeWithNetworks(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name: "myapp",
		Networks: map[string]types.NetworkConfig{
			"backend": {
				Name:   "backend",
				Driver: "bridge",
			},
		},
		Services: map[string]types.ServiceConfig{
			"db": {
				Name:        "db",
				Image:       "postgres:15",
				NetworkMode: "bridge",
				Networks: map[string]*types.ServiceNetworkConfig{
					"backend": {},
				},
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// Even with explicit bridge mode, service should have network in ServiceNetworks
	assert.Equal(t, "bridge", spec.Container.Network.Mode)
	assert.Contains(t, spec.Container.Network.ServiceNetworks, "myapp-backend")
}

// TestNetworkDependencies_NoNetworksInProject tests behavior when project
// has no networks defined (edge case).
func TestNetworkDependencies_NoNetworksInProject(t *testing.T) {
	converter := NewSpecConverter(".")

	project := &types.Project{
		Name:     "myapp",
		Networks: map[string]types.NetworkConfig{}, // No networks
		Services: map[string]types.ServiceConfig{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
			},
		},
	}

	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	spec := specs[0]

	// When project has no networks, ServiceNetworks should be empty
	// (Podman will use default bridge network implicitly)
	assert.Empty(t, spec.Container.Network.ServiceNetworks)
	assert.Empty(t, spec.Networks)
}
