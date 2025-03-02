package unit

// NetworkConfig represents the configuration for a network in a Quadlet unit.
type NetworkConfig struct {
	Label      []string `yaml:"label"`
	Driver     string   `yaml:"driver"`
	Gateway    string   `yaml:"gateway"`
	IPRange    string   `yaml:"ip_range"`
	Subnet     string   `yaml:"subnet"`
	IPv6       bool     `yaml:"ipv6"`
	Internal   bool     `yaml:"internal"`
	DNSEnabled bool     `yaml:"dns_enabled"`
	Options    []string `yaml:"options"`
}
