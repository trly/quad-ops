package unit

// Volume represents the configuration for a volume in a Quadlet unit.
type Volume struct {
	ContainersConfModule []string `yaml:"containers_conf_module"`
	Copy                 bool     `yaml:"copy"`
	Device               string   `yaml:"device"`
	Driver               string   `yaml:"driver"`
	GlobalArgs           []string `yaml:"global_args"`
	Group                string   `yaml:"group"`
	Image                string   `yaml:"image"`
	Label                []string `yaml:"label"`
	Options              []string `yaml:"options"`
	PodmanArgs           []string `yaml:"podman_args"`
	Type                 string   `yaml:"type"`
	User                 string   `yaml:"user"`
	VolumeName           string   `yaml:"volume_name"`

	// Systemd unit properties
	Name     string
	UnitType string
}

// NewVolume creates a new Volume with the given name
func NewVolume(name string) *Volume {
	return &Volume{
		Name:     name,
		UnitType: "volume",
	}
}

// GetServiceName returns the full systemd service name
func (v *Volume) GetServiceName() string {
	return v.Name + "-volume.service"
}

// GetUnitType returns the type of the unit
func (v *Volume) GetUnitType() string {
	return "volume"
}

// GetUnitName returns the name of the unit
func (v *Volume) GetUnitName() string {
	return v.Name
}

// GetStatus returns the current status of the unit
func (v *Volume) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: v.Name, Type: "volume"}
	return base.GetStatus()
}

// Start starts the unit
func (v *Volume) Start() error {
	base := BaseSystemdUnit{Name: v.Name, Type: "volume"}
	return base.Start()
}

// Stop stops the unit
func (v *Volume) Stop() error {
	base := BaseSystemdUnit{Name: v.Name, Type: "volume"}
	return base.Stop()
}

// Restart restarts the unit
func (v *Volume) Restart() error {
	base := BaseSystemdUnit{Name: v.Name, Type: "volume"}
	return base.Restart()
}

// Show displays the unit configuration and status
func (v *Volume) Show() error {
	base := BaseSystemdUnit{Name: v.Name, Type: "volume"}
	return base.Show()
}
