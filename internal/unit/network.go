package unit

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/systemd"
)

// Network represents the configuration for a network in a Quadlet unit.
type Network struct {
	BaseUnit          // Embed the base struct
	Label    []string `yaml:"label"`
	Driver   string   `yaml:"driver"`
	Gateway  string   `yaml:"gateway"`
	IPRange  string   `yaml:"ip_range"`
	Subnet   string   `yaml:"subnet"`
	IPv6     bool     `yaml:"ipv6"`
	Internal bool     `yaml:"internal"`
	// DNSEnabled removed - not supported by podman-systemd
	Options     []string `yaml:"options"`
	NetworkName string   `yaml:"network_name"`
}

// NewNetwork creates a new Network with the given name.
func NewNetwork(name string) *Network {
	return &Network{
		BaseUnit: BaseUnit{
			BaseUnit: systemd.NewBaseUnit(name, "network"),
			Name:     name,
			UnitType: "network",
		},
	}
}

// FromComposeNetwork creates a Network from a Docker Compose network configuration.
func (n *Network) FromComposeNetwork(name string, network types.NetworkConfig) *Network {
	// Set network name if specified in compose file, otherwise use the key name
	if network.Name != "" {
		n.NetworkName = network.Name
	} else {
		n.NetworkName = name
	}

	// Set driver if specified
	if network.Driver != "" {
		n.Driver = network.Driver
	}

	// Handle IPAM configuration if present
	if len(network.Ipam.Config) > 0 {
		// Use the first IPAM pool configuration
		config := network.Ipam.Config[0]

		if config.Subnet != "" {
			n.Subnet = config.Subnet
		}

		if config.Gateway != "" {
			n.Gateway = config.Gateway
		}

		if config.IPRange != "" {
			n.IPRange = config.IPRange
		}
	}

	// Set internal flag
	if network.Internal {
		n.Internal = true
	}

	// Set IPv6 flag if enabled
	if network.EnableIPv6 != nil && *network.EnableIPv6 {
		n.IPv6 = true
	}

	// Convert driver options to options array
	if len(network.DriverOpts) > 0 {
		for key, value := range network.DriverOpts {
			n.Options = append(n.Options, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Add labels
	if len(network.Labels) > 0 {
		n.Label = append(n.Label, network.Labels.AsList()...)
	}

	// DNS is configured via driver options in podman-systemd

	// Sort all slices for deterministic output
	sortNetwork(n)

	return n
}

// sortNetwork ensures all slices in a network config are sorted deterministically in-place.
func sortNetwork(n *Network) {
	// Sort all slices for deterministic output
	if len(n.Label) > 0 {
		sort.Strings(n.Label)
	}

	if len(n.Options) > 0 {
		sort.Strings(n.Options)
	}
}
