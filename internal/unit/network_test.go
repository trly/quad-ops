package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestFromComposeNetwork(t *testing.T) {
	// Test case: Network with complete configuration
	networkName := "test-network"
	// Setup IPv6 flag
	ipv6Enabled := true

	// Create a compose network config with all settings
	composeNetwork := types.NetworkConfig{
		Name:       "custom-network-name",
		Driver:     "bridge",
		Internal:   true,
		EnableIPv6: &ipv6Enabled,
		DriverOpts: map[string]string{
			"com.docker.network.bridge.name":                 "custom-bridge",
			"com.docker.network.bridge.enable_icc":          "true",
			"com.docker.network.bridge.enable_ip_masquerade": "true",
		},
		Labels: types.Labels{
			"com.example.description": "Test network",
			"com.example.department":  "IT",
		},
		Ipam: types.IPAMConfig{
			Driver: "default",
			Config: []*types.IPAMPool{
				{
					Subnet:  "172.28.0.0/16",
					Gateway: "172.28.0.1",
					IPRange: "172.28.5.0/24",
				},
			},
		},
	}

	network := NewNetwork(networkName)
	network = network.FromComposeNetwork(networkName, composeNetwork)

	// Verify the conversion was correct
	// Basic properties
	assert.Equal(t, networkName, network.Name)
	assert.Equal(t, "network", network.UnitType)

	// Network driver
	assert.Equal(t, "bridge", network.Driver)
	
	// Network flags
	assert.True(t, network.Internal)
	assert.True(t, network.IPv6)
	// DNSEnabled is not supported by podman-systemd

	// IPAM config
	assert.Equal(t, "172.28.0.0/16", network.Subnet)
	assert.Equal(t, "172.28.0.1", network.Gateway)
	assert.Equal(t, "172.28.5.0/24", network.IPRange)

	// Driver options
	assert.Contains(t, network.Options, "com.docker.network.bridge.name=custom-bridge")
	assert.Contains(t, network.Options, "com.docker.network.bridge.enable_icc=true")
	assert.Contains(t, network.Options, "com.docker.network.bridge.enable_ip_masquerade=true")

	// Labels
	assert.Contains(t, network.Label, "com.example.description=Test network")
	assert.Contains(t, network.Label, "com.example.department=IT")

	// Test case: Minimal network configuration
	minimalNetworkName := "minimal-network"
	minimalComposeNetwork := types.NetworkConfig{}

	minimalNetwork := NewNetwork(minimalNetworkName)
	minimalNetwork = minimalNetwork.FromComposeNetwork(minimalNetworkName, minimalComposeNetwork)

	// Verify minimal network configuration
	assert.Equal(t, minimalNetworkName, minimalNetwork.Name)
	assert.Equal(t, "network", minimalNetwork.UnitType)
	
	// Default values
	// DNSEnabled is not supported by podman-systemd
	assert.Empty(t, minimalNetwork.Driver, "Driver should be empty for minimal configuration")
	assert.Empty(t, minimalNetwork.Subnet, "Subnet should be empty for minimal configuration")
}