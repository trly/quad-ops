package systemd

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/ini.v1"
)

// BuildNetwork converts a compose network into a network unit file.
func BuildNetwork(projectName, netName string, net *types.NetworkConfig) Unit {
	file := ini.Empty(ini.LoadOptions{AllowShadows: true})
	section, _ := file.NewSection("Network")
	sectionMap := make(map[string]string)
	shadowMap := make(map[string][]string) // For keys with repeated values
	buildNetworkSection(netName, net, sectionMap, shadowMap)
	writeOrderedSection(section, sectionMap, shadowMap)

	return Unit{
		Name: fmt.Sprintf("%s-%s.network", projectName, netName),
		File: file,
	}
}

func buildNetworkSection(_ string, net *types.NetworkConfig, section map[string]string, shadows map[string][]string) {
	// Driver defaults to bridge if not specified
	if net.Driver != "" {
		section["Driver"] = net.Driver
	}

	// NetworkName: custom name or defaults to systemd-$name
	if net.Name != "" {
		section["NetworkName"] = net.Name
	}

	// Labels: map compose labels to systemd Label= directives
	// Uses dot-notation for multi-value serialization: Label.key=value
	for k, v := range net.Labels {
		section[fmt.Sprintf("Label.%s", k)] = v
	}

	// Internal: restrict external access
	if net.Internal {
		section["Internal"] = "true"
	}

	// EnableIPv6: enable dual-stack networking
	if net.EnableIPv6 != nil && *net.EnableIPv6 {
		section["IPv6"] = "true"
	}

	// DriverOpts mapping to Podman systemd directives
	if len(net.DriverOpts) > 0 {
		mapNetworkDriverOpts(net.DriverOpts, section, shadows)
	}

	// IPAM configuration mapping
	if len(net.Ipam.Config) > 0 {
		mapIPAMConfig(net.Ipam.Config, section)
	}

	// x-quad-ops-podman-args: list of global podman arguments
	if globalArgs, ok := net.Extensions["x-quad-ops-podman-args"].([]interface{}); ok {
		for _, arg := range globalArgs {
			if argStr, ok := arg.(string); ok {
				shadows["PodmanArgs"] = append(shadows["PodmanArgs"], argStr)
			}
		}
	}

	// x-quad-ops-network-args: list of network-specific podman arguments
	if networkArgs, ok := net.Extensions["x-quad-ops-network-args"].([]interface{}); ok {
		for _, arg := range networkArgs {
			if argStr, ok := arg.(string); ok {
				shadows["PodmanArgs"] = append(shadows["PodmanArgs"], argStr)
			}
		}
	}
}

// mapNetworkDriverOpts maps compose driver options to Podman systemd [Network] directives.
// See: https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#network-units-network
func mapNetworkDriverOpts(opts map[string]string, section map[string]string, shadows map[string][]string) {
	for k, v := range opts {
		switch k {
		case "disable_dns":
			// DisableDNS=true → --disable-dns
			if v == "true" {
				section["DisableDNS"] = "true"
			}

		case "dns":
			// DNS=192.168.55.1 → --dns=192.168.55.1
			// Handled separately as DNS can be repeated
			shadows["DNS"] = append(shadows["DNS"], v)

		case "gateway":
			// Gateway=192.168.55.3 → --gateway 192.168.55.3
			section["Gateway"] = v

		case "interface_name":
			// InterfaceName=enp1 → --interface-name enp1
			section["InterfaceName"] = v

		case "internal":
			// Internal=true → --internal
			if v == "true" {
				section["Internal"] = "true"
			}

		case "ipam_driver":
			// IPAMDriver=dhcp → --ipam-driver dhcp
			section["IPAMDriver"] = v

		case "ip_range":
			// IPRange=192.168.55.128/25 → --ip-range 192.168.55.128/25
			// IPRange can be repeated for multiple ranges
			shadows["IPRange"] = append(shadows["IPRange"], v)

		case "ipv6":
			// IPv6=true → --ipv6
			if v == "true" {
				section["IPv6"] = "true"
			}

		case "options", "opt":
			// Options=isolate=true → --opt isolate=true
			section["Options"] = v

		case "subnet":
			// Subnet=192.5.0.0/16 → --subnet 192.5.0.0/16
			section["Subnet"] = v

		case "module", "containers-conf-module":
			section["ContainersConfModule"] = v

		// Network-specific boolean options without values
		case "network_delete_on_stop":
			if v == "true" {
				section["NetworkDeleteOnStop"] = "true"
			}

		default:
			// Ignore unknown driver options to avoid polluting the section
			// with compose-specific or driver-specific settings
		}
	}
}

// mapIPAMConfig maps compose IPAM pool configuration to systemd directives.
func mapIPAMConfig(ipamPools []*types.IPAMPool, section map[string]string) {
	for i, pool := range ipamPools {
		if pool == nil {
			continue
		}
		if pool.Subnet != "" {
			section[fmt.Sprintf("Subnet.%d", i)] = pool.Subnet
		}
		if pool.Gateway != "" {
			section[fmt.Sprintf("Gateway.%d", i)] = pool.Gateway
		}
		if pool.IPRange != "" {
			section[fmt.Sprintf("IPRange.%d", i)] = pool.IPRange
		}
	}
}
