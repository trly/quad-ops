package model

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// Network represents the configuration for a network in a Quadlet unit.
type Network struct {
	Name       string
	UnitType   string
	Label      []string `yaml:"label"`
	Driver     string   `yaml:"driver"`
	Gateway    string   `yaml:"gateway"`
	IPRange    string   `yaml:"ip_range"`
	Subnet     string   `yaml:"subnet"`
	IPv6       bool     `yaml:"ipv6"`
	Internal   bool     `yaml:"internal"`
	Options    []string `yaml:"options"`
	NetworkName string   `yaml:"network_name"`
}

// NewNetwork creates a new Network with the given name.
func NewNetwork(name string) *Network {
	return &Network{
		Name:     name,
		UnitType: "network",
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

	return n
}

// GetServiceName returns the full systemd service name.
func (n *Network) GetServiceName() string {
	return n.Name + "-network.service"
}

// GetUnitType returns the type of the unit.
func (n *Network) GetUnitType() string {
	return "network"
}

// GetUnitName returns the name of the unit.
func (n *Network) GetUnitName() string {
	return n.Name
}