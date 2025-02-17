package quadlet

// QuadletUnit represents a unit configuration for Quadlet, containing the unit name,
// type, systemd configuration, and additional key-value configuration settings.
type QuadletUnit struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Systemd SystemdConfig     `yaml:"systemd"`
	Config  map[string]string `yaml:"config"`
}

// SystemdConfig represents the systemd-specific configuration options for a Quadlet unit,
// including the unit description, dependencies, and restart behavior.
type SystemdConfig struct {
	Description   string   `yaml:"description"`
	After         []string `yaml:"after"`
	RestartPolicy string   `yaml:"restart_policy"`
}
