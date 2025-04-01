package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNetworkConfigYAMLMarshaling(t *testing.T) {
	// Create a sample network config
	config := Network{
		Label:      []string{"app=web", "environment=production"},
		Driver:     "bridge",
		Gateway:    "192.168.0.1",
		IPRange:    "192.168.0.0/24",
		Subnet:     "192.168.0.0/16",
		IPv6:       true,
		Internal:   false,
		DNSEnabled: true,
		Options:    []string{"com.example.option1=value1", "com.example.option2=value2"},
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Network
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, config.Driver, unmarshaled.Driver)
	assert.Equal(t, config.Gateway, unmarshaled.Gateway)
	assert.Equal(t, config.IPRange, unmarshaled.IPRange)
	assert.Equal(t, config.Subnet, unmarshaled.Subnet)
	assert.Equal(t, config.IPv6, unmarshaled.IPv6)
	assert.Equal(t, config.Internal, unmarshaled.Internal)
	assert.Equal(t, config.DNSEnabled, unmarshaled.DNSEnabled)
	assert.ElementsMatch(t, config.Label, unmarshaled.Label)
	assert.ElementsMatch(t, config.Options, unmarshaled.Options)
}

func TestNetworkConfigYAMLUnmarshaling(t *testing.T) {
	yamlData := `
label:
  - project=myapp
  - tier=backend
driver: macvlan
gateway: 10.0.0.1
ip_range: 10.0.0.0/24
subnet: 10.0.0.0/16
ipv6: false
internal: true
dns_enabled: false
options:
  - parent=eth0
  - mtu=1500
`

	var config Network
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)

	// Verify fields were properly unmarshaled
	assert.ElementsMatch(t, []string{"project=myapp", "tier=backend"}, config.Label)
	assert.Equal(t, "macvlan", config.Driver)
	assert.Equal(t, "10.0.0.1", config.Gateway)
	assert.Equal(t, "10.0.0.0/24", config.IPRange)
	assert.Equal(t, "10.0.0.0/16", config.Subnet)
	assert.False(t, config.IPv6)
	assert.True(t, config.Internal)
	assert.False(t, config.DNSEnabled)
	assert.ElementsMatch(t, []string{"parent=eth0", "mtu=1500"}, config.Options)
}

func TestNetworkConfigYAMLMarshalingPartial(t *testing.T) {
	// Test with only some fields populated
	config := Network{
		Driver: "host",
		IPv6:   true,
		// Other fields left empty
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Network
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, "host", unmarshaled.Driver)
	assert.True(t, unmarshaled.IPv6)
	assert.Empty(t, unmarshaled.Label)
	assert.Empty(t, unmarshaled.Gateway)
	assert.Empty(t, unmarshaled.IPRange)
	assert.Empty(t, unmarshaled.Subnet)
	assert.False(t, unmarshaled.Internal)
	assert.False(t, unmarshaled.DNSEnabled)
	assert.Empty(t, unmarshaled.Options)
}
