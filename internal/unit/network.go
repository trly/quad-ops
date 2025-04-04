package unit

// Network represents the configuration for a network in a Quadlet unit.
type Network struct {
	Label      []string `yaml:"label"`
	Driver     string   `yaml:"driver"`
	Gateway    string   `yaml:"gateway"`
	IPRange    string   `yaml:"ip_range"`
	Subnet     string   `yaml:"subnet"`
	IPv6       bool     `yaml:"ipv6"`
	Internal   bool     `yaml:"internal"`
	DNSEnabled bool     `yaml:"dns_enabled"`
	Options    []string `yaml:"options"`

	// Systemd unit properties
	Name     string
	UnitType string
}

// NewNetwork creates a new Network with the given name
func NewNetwork(name string) *Network {
	return &Network{
		Name:     name,
		UnitType: "network",
	}
}

// GetServiceName returns the full systemd service name
func (n *Network) GetServiceName() string {
	return n.Name + "-network.service"
}

// GetUnitType returns the type of the unit
func (n *Network) GetUnitType() string {
	return "network"
}

// GetUnitName returns the name of the unit
func (n *Network) GetUnitName() string {
	return n.Name
}

// GetStatus returns the current status of the unit
func (n *Network) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: n.Name, Type: "network"}
	return base.GetStatus()
}

// Start starts the unit
func (n *Network) Start() error {
	base := BaseSystemdUnit{Name: n.Name, Type: "network"}
	return base.Start()
}

// Stop stops the unit
func (n *Network) Stop() error {
	base := BaseSystemdUnit{Name: n.Name, Type: "network"}
	return base.Stop()
}

// Restart restarts the unit
func (n *Network) Restart() error {
	base := BaseSystemdUnit{Name: n.Name, Type: "network"}
	return base.Restart()
}

// Show displays the unit configuration and status
func (n *Network) Show() error {
	base := BaseSystemdUnit{Name: n.Name, Type: "network"}
	return base.Show()
}
