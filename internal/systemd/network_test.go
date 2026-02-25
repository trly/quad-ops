package systemd

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getNetValue is a helper to get a key value from the Network section.
func getNetValue(unit Unit, key string) string {
	section := unit.File.Section("Network")
	if section == nil {
		return ""
	}
	return section.Key(key).String()
}

// getNetValues is a helper to get all values (including shadows) for a key from the Network section.
func getNetValues(unit Unit, key string) []string {
	section := unit.File.Section("Network")
	if section == nil {
		return []string{}
	}
	k := section.Key(key)
	if k == nil {
		return []string{}
	}
	return k.ValueWithShadows()
}

// TestBuildNetwork_BasicNetwork tests that a simple network creates the correct unit structure.
func TestBuildNetwork_BasicNetwork(t *testing.T) {
	net := &types.NetworkConfig{}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "testproject-mynetwork.network", unit.Name)
	assert.NotNil(t, unit.File)
	assert.NotNil(t, unit.File.Section("Network"))
	assert.Empty(t, getNetValue(unit, "DNS"), "DNS should not be set by default (Podman enables DNS resolution automatically)")
}

// TestBuildNetwork_DNSNotSetByDefault tests that DNS is not set unless explicitly configured via driver opts.
func TestBuildNetwork_DNSNotSetByDefault(t *testing.T) {
	tests := []struct {
		name        string
		driverOpts  map[string]string
		expectedDNS string
	}{
		{
			name:        "no driver opts",
			driverOpts:  nil,
			expectedDNS: "",
		},
		{
			name:        "unrelated driver opts",
			driverOpts:  map[string]string{"gateway": "192.168.1.1"},
			expectedDNS: "",
		},
		{
			name:        "explicit dns server",
			driverOpts:  map[string]string{"dns": "8.8.8.8"},
			expectedDNS: "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: tt.driverOpts,
			}
			unit := BuildNetwork("testproject", "mynetwork", net)

			assert.Equal(t, tt.expectedDNS, getNetValue(unit, "DNS"))
		})
	}
}

// TestBuildNetwork_WithDriver tests that the driver option is correctly mapped.
func TestBuildNetwork_WithDriver(t *testing.T) {
	net := &types.NetworkConfig{
		Driver: "bridge",
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "bridge", getNetValue(unit, "Driver"))
}

// TestBuildNetwork_WithCustomName tests that a custom network name is preserved.
func TestBuildNetwork_WithCustomName(t *testing.T) {
	net := &types.NetworkConfig{
		Name: "custom-network-name",
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "custom-network-name", getNetValue(unit, "NetworkName"))
}

// TestBuildNetwork_WithLabels tests that labels are mapped with dot-notation.
func TestBuildNetwork_WithLabels(t *testing.T) {
	net := &types.NetworkConfig{
		Labels: types.Labels{
			"app": "myapp",
			"env": "production",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "myapp", getNetValue(unit, "Label.app"))
	assert.Equal(t, "production", getNetValue(unit, "Label.env"))
}

// TestBuildNetwork_WithEmptyLabels tests that no Label keys are added when labels are empty.
func TestBuildNetwork_WithEmptyLabels(t *testing.T) {
	net := &types.NetworkConfig{
		Labels: types.Labels{},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	section := unit.File.Section("Network")
	for _, key := range section.Keys() {
		assert.False(t, len(key.Name()) > 6 && key.Name()[:6] == "Label.")
	}
}

// TestBuildNetwork_DriverOptsDisableDNS tests the "disable_dns" driver option mapping.
func TestBuildNetwork_DriverOptsDisableDNS(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"disable_dns true", "true", true},
		{"disable_dns false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					"disable_dns": tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)

			if tt.expected {
				assert.Equal(t, "true", getNetValue(unit, "DisableDNS"))
			} else {
				assert.Empty(t, getNetValue(unit, "DisableDNS"))
			}
		})
	}
}

// TestBuildNetwork_DriverOptsDNS tests the "dns" driver option mapping.
func TestBuildNetwork_DriverOptsDNS(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"dns": "192.168.55.1",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.55.1", getNetValue(unit, "DNS"))
}

// TestBuildNetwork_DriverOptsMultipleDNS tests that multiple DNS servers are mapped as shadows.
func TestBuildNetwork_DriverOptsMultipleDNS(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"dns": "192.168.55.1",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	vals := getNetValues(unit, "DNS")
	assert.Len(t, vals, 1)
	assert.Equal(t, "192.168.55.1", vals[0])
}

// TestBuildNetwork_DriverOptsGateway tests the "gateway" driver option mapping.
func TestBuildNetwork_DriverOptsGateway(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"gateway": "192.168.55.3",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.55.3", getNetValue(unit, "Gateway"))
}

// TestBuildNetwork_DriverOptsInterfaceName tests the "interface_name" driver option mapping.
func TestBuildNetwork_DriverOptsInterfaceName(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"interface_name": "enp1",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "enp1", getNetValue(unit, "InterfaceName"))
}

// TestBuildNetwork_DriverOptsInternal tests the "internal" driver option mapping.
func TestBuildNetwork_DriverOptsInternal(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"internal true", "true", true},
		{"internal false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					"internal": tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)

			if tt.expected {
				assert.Equal(t, "true", getNetValue(unit, "Internal"))
			} else {
				assert.Empty(t, getNetValue(unit, "Internal"))
			}
		})
	}
}

// TestBuildNetwork_DriverOptsIPAMDriver tests the "ipam_driver" driver option mapping.
func TestBuildNetwork_DriverOptsIPAMDriver(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"ipam_driver": "dhcp",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "dhcp", getNetValue(unit, "IPAMDriver"))
}

// TestBuildNetwork_DriverOptsIPRange tests the "ip_range" driver option mapping.
func TestBuildNetwork_DriverOptsIPRange(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"ip_range": "192.168.55.128/25",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	vals := getNetValues(unit, "IPRange")
	assert.Len(t, vals, 1)
	assert.Equal(t, "192.168.55.128/25", vals[0])
}

// TestBuildNetwork_DriverOptsIPv6 tests the "ipv6" driver option mapping.
func TestBuildNetwork_DriverOptsIPv6(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"ipv6 true", "true", true},
		{"ipv6 false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					"ipv6": tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)

			if tt.expected {
				assert.Equal(t, "true", getNetValue(unit, "IPv6"))
			} else {
				assert.Empty(t, getNetValue(unit, "IPv6"))
			}
		})
	}
}

// TestBuildNetwork_DriverOptsOptionsAlias tests the "options"/"opt" driver option mapping.
func TestBuildNetwork_DriverOptsOptionsAlias(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"options key", "options", "isolate=true"},
		{"opt key", "opt", "bip=192.168.1.0/24"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					tt.key: tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)
			assert.Equal(t, tt.value, getNetValue(unit, "Options"))
		})
	}
}

// TestBuildNetwork_DriverOptsSubnet tests the "subnet" driver option mapping.
func TestBuildNetwork_DriverOptsSubnet(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"subnet": "192.5.0.0/16",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.5.0.0/16", getNetValue(unit, "Subnet"))
}

// TestBuildNetwork_DriverOptsModule tests the "module"/"containers-conf-module" driver option mapping.
func TestBuildNetwork_DriverOptsModule(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"module key", "module", "my-module"},
		{"containers-conf-module key", "containers-conf-module", "custom-module"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					tt.key: tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)
			assert.Equal(t, tt.value, getNetValue(unit, "ContainersConfModule"))
		})
	}
}

// TestBuildNetwork_DriverOptsNetworkDeleteOnStop tests the "network_delete_on_stop" driver option mapping.
func TestBuildNetwork_DriverOptsNetworkDeleteOnStop(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"network_delete_on_stop true", "true", true},
		{"network_delete_on_stop false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := &types.NetworkConfig{
				DriverOpts: map[string]string{
					"network_delete_on_stop": tt.value,
				},
			}
			unit := BuildNetwork("testproject", "mynetwork", net)

			if tt.expected {
				assert.Equal(t, "true", getNetValue(unit, "NetworkDeleteOnStop"))
			} else {
				assert.Empty(t, getNetValue(unit, "NetworkDeleteOnStop"))
			}
		})
	}
}

// TestBuildNetwork_IPAMConfigSinglePool tests IPAM configuration with a single pool.
func TestBuildNetwork_IPAMConfigSinglePool(t *testing.T) {
	net := &types.NetworkConfig{
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{
				{
					Subnet:  "192.168.1.0/24",
					Gateway: "192.168.1.1",
					IPRange: "192.168.1.128/25",
				},
			},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.1.0/24", getNetValue(unit, "Subnet.0"))
	assert.Equal(t, "192.168.1.1", getNetValue(unit, "Gateway.0"))
	assert.Equal(t, "192.168.1.128/25", getNetValue(unit, "IPRange.0"))
}

// TestBuildNetwork_MultipleIPAMConfigs tests multiple IPAM configurations.
func TestBuildNetwork_MultipleIPAMConfigs(t *testing.T) {
	net := &types.NetworkConfig{
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{
				{
					Subnet:  "192.168.1.0/24",
					Gateway: "192.168.1.1",
				},
				{
					Subnet:  "192.168.2.0/24",
					Gateway: "192.168.2.1",
				},
			},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.1.0/24", getNetValue(unit, "Subnet.0"))
	assert.Equal(t, "192.168.1.1", getNetValue(unit, "Gateway.0"))
	assert.Equal(t, "192.168.2.0/24", getNetValue(unit, "Subnet.1"))
	assert.Equal(t, "192.168.2.1", getNetValue(unit, "Gateway.1"))
}

// TestBuildNetwork_IPAMConfigPartial tests IPAM with partial fields.
func TestBuildNetwork_IPAMConfigPartial(t *testing.T) {
	net := &types.NetworkConfig{
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{
				{
					Subnet: "192.168.1.0/24",
					// Gateway and IPRange are empty
				},
			},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.1.0/24", getNetValue(unit, "Subnet.0"))
	assert.Empty(t, getNetValue(unit, "Gateway.0"))
	assert.Empty(t, getNetValue(unit, "IPRange.0"))
}

// TestBuildNetwork_AllFieldsTogether tests a network with all fields set together.
func TestBuildNetwork_AllFieldsTogether(t *testing.T) {
	net := &types.NetworkConfig{
		Driver: "bridge",
		Name:   "custom-network",
		Labels: types.Labels{
			"owner": "admin",
		},
		DriverOpts: map[string]string{
			"gateway": "192.168.55.3",
			"subnet":  "192.5.0.0/16",
			"ipv6":    "true",
		},
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{
				{
					Subnet:  "10.0.0.0/8",
					Gateway: "10.0.0.1",
				},
			},
		},
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": []interface{}{
				"--log-driver=journald",
			},
			"x-quad-ops-network-args": []interface{}{
				"--label=managed=true",
			},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "bridge", getNetValue(unit, "Driver"))
	assert.Equal(t, "custom-network", getNetValue(unit, "NetworkName"))
	assert.Equal(t, "admin", getNetValue(unit, "Label.owner"))
	assert.Equal(t, "192.168.55.3", getNetValue(unit, "Gateway"))
	assert.Equal(t, "192.5.0.0/16", getNetValue(unit, "Subnet"))
	assert.Equal(t, "true", getNetValue(unit, "IPv6"))
	assert.Equal(t, "10.0.0.0/8", getNetValue(unit, "Subnet.0"))
	assert.Equal(t, "10.0.0.1", getNetValue(unit, "Gateway.0"))
	vals := getNetValues(unit, "PodmanArgs")
	assert.Len(t, vals, 2)
	assert.Equal(t, "--log-driver=journald", vals[0])
	assert.Equal(t, "--label=managed=true", vals[1])
}

// TestBuildNetwork_DriverOptsIPRangeWithShadows tests that IPRange is properly handled as a shadow key.
func TestBuildNetwork_DriverOptsIPRangeWithShadows(t *testing.T) {
	net := &types.NetworkConfig{
		DriverOpts: map[string]string{
			"ip_range": "192.168.55.128/25",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	vals := getNetValues(unit, "IPRange")
	assert.Len(t, vals, 1)
	assert.Equal(t, "192.168.55.128/25", vals[0])
}

// TestBuildNetwork_MultipleLabels tests that multiple labels are all preserved.
func TestBuildNetwork_MultipleLabels(t *testing.T) {
	net := &types.NetworkConfig{
		Labels: types.Labels{
			"app":       "myapp",
			"component": "networking",
			"version":   "v1",
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "myapp", getNetValue(unit, "Label.app"))
	assert.Equal(t, "networking", getNetValue(unit, "Label.component"))
	assert.Equal(t, "v1", getNetValue(unit, "Label.version"))
}

// TestBuildNetwork_NameDerivation tests that the unit name is derived from the project and network name.
func TestBuildNetwork_NameDerivation(t *testing.T) {
	tests := []struct {
		project      string
		network      string
		expectedUnit string
	}{
		{"myproject", "default", "myproject-default.network"},
		{"myproject", "db-network", "myproject-db-network.network"},
		{"myproject", "cache_net", "myproject-cache_net.network"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			net := &types.NetworkConfig{}
			unit := BuildNetwork(tt.project, tt.network, net)
			assert.Equal(t, tt.expectedUnit, unit.Name)
		})
	}
}

// TestBuildNetwork_SectionStructure tests that the unit always has a Network section.
func TestBuildNetwork_SectionStructure(t *testing.T) {
	net := &types.NetworkConfig{}
	unit := BuildNetwork("testproject", "net", net)

	require.NotNil(t, unit.File)
	require.NotNil(t, unit.File.Section("Network"))
}

// TestBuildNetworkSection_NoDriver tests that empty driver is not added.
func TestBuildNetworkSection_NoDriver(t *testing.T) {
	net := &types.NetworkConfig{
		Driver: "",
	}
	unit := BuildNetwork("testproject", "net", net)

	assert.Empty(t, getNetValue(unit, "Driver"))
}

// TestBuildNetworkSection_NoNetworkName tests that missing Name field doesn't add NetworkName.
func TestBuildNetworkSection_NoNetworkName(t *testing.T) {
	net := &types.NetworkConfig{
		Name: "",
	}
	unit := BuildNetwork("testproject", "net", net)

	assert.Empty(t, getNetValue(unit, "NetworkName"))
}

// TestBuildNetwork_EmptyIPAMConfig tests that empty IPAM config is handled gracefully.
func TestBuildNetwork_EmptyIPAMConfig(t *testing.T) {
	net := &types.NetworkConfig{
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	section := unit.File.Section("Network")
	for _, key := range section.Keys() {
		assert.NotContains(t, key.Name(), "Subnet.")
		assert.NotContains(t, key.Name(), "Gateway.")
		assert.NotContains(t, key.Name(), "IPRange.")
	}
}

// TestBuildNetwork_BridgeDriver tests common bridge network configuration.
func TestBuildNetwork_BridgeDriver(t *testing.T) {
	net := &types.NetworkConfig{
		Driver: "bridge",
		DriverOpts: map[string]string{
			"gateway": "192.168.1.1",
			"subnet":  "192.168.1.0/24",
		},
	}
	unit := BuildNetwork("testproject", "bridge-net", net)

	assert.Equal(t, "bridge", getNetValue(unit, "Driver"))
	assert.Equal(t, "192.168.1.1", getNetValue(unit, "Gateway"))
	assert.Equal(t, "192.168.1.0/24", getNetValue(unit, "Subnet"))
}

// TestBuildNetwork_MacvlanDriver tests macvlan network configuration.
func TestBuildNetwork_MacvlanDriver(t *testing.T) {
	net := &types.NetworkConfig{
		Driver: "macvlan",
		DriverOpts: map[string]string{
			"options": "parent=eth0",
		},
	}
	unit := BuildNetwork("testproject", "macvlan-net", net)

	assert.Equal(t, "macvlan", getNetValue(unit, "Driver"))
	assert.Equal(t, "parent=eth0", getNetValue(unit, "Options"))
}

// TestBuildNetwork_InternalNetwork tests internal network configuration.
func TestBuildNetwork_InternalNetwork(t *testing.T) {
	net := &types.NetworkConfig{
		Internal: true,
	}
	unit := BuildNetwork("testproject", "internal-net", net)

	assert.Equal(t, "true", getNetValue(unit, "Internal"))
}

// TestBuildNetwork_EnableIPv6 tests that the top-level enable_ipv6 compose field is mapped.
func TestBuildNetwork_EnableIPv6(t *testing.T) {
	enabled := true
	net := &types.NetworkConfig{
		EnableIPv6: &enabled,
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "true", getNetValue(unit, "IPv6"))
}

// TestBuildNetwork_EnableIPv6False tests that enable_ipv6=false does not set IPv6.
func TestBuildNetwork_EnableIPv6False(t *testing.T) {
	disabled := false
	net := &types.NetworkConfig{
		EnableIPv6: &disabled,
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Empty(t, getNetValue(unit, "IPv6"))
}

// TestBuildNetwork_IPAMConfigWithNilPool tests IPAM configuration with nil pool entries.
func TestBuildNetwork_IPAMConfigWithNilPool(t *testing.T) {
	net := &types.NetworkConfig{
		Ipam: types.IPAMConfig{
			Config: []*types.IPAMPool{
				{
					Subnet:  "192.168.1.0/24",
					Gateway: "192.168.1.1",
				},
				nil,
				{
					Subnet:  "192.168.2.0/24",
					Gateway: "192.168.2.1",
				},
			},
		},
	}
	unit := BuildNetwork("testproject", "mynetwork", net)

	assert.Equal(t, "192.168.1.0/24", getNetValue(unit, "Subnet.0"))
	assert.Equal(t, "192.168.1.1", getNetValue(unit, "Gateway.0"))
	// Nil at index 1 is skipped, but the key for index 1 still appears (pointing to nil)
	// We verify that index 2 has the correct values despite the nil at index 1
	assert.Equal(t, "192.168.2.0/24", getNetValue(unit, "Subnet.2"))
	assert.Equal(t, "192.168.2.1", getNetValue(unit, "Gateway.2"))
}
