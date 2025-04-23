package model

// SystemdUnit defines the interface for systemd unit operations.
type SystemdUnit interface {
	GetServiceName() string
	GetUnitType() string
	GetUnitName() string
}

// QuadletUnitConfig represents the configuration for a Quadlet unit with different types.
type QuadletUnitConfig struct {
	Name      string
	Type      string
	Systemd   SystemdConfig
	Container Container `yaml:"container,omitempty"`
	Volume    Volume    `yaml:"volume,omitempty"`
	Network   Network   `yaml:"network,omitempty"`
}

// SystemdConfig represents the configuration for a systemd unit.
type SystemdConfig struct {
	Description        string   `yaml:"description"`
	After              []string `yaml:"after"`
	Before             []string `yaml:"before"`
	Requires           []string `yaml:"requires"`
	Wants              []string `yaml:"wants"`
	Conflicts          []string `yaml:"conflicts"`
	PartOf             []string `yaml:"part_of"`
	PropagatesReloadTo []string `yaml:"propagates_reload_to"`
	RestartPolicy      string   `yaml:"restart_policy"`
	TimeoutStartSec    int      `yaml:"timeout_start_sec"`
	Type               string   `yaml:"type"`
	RemainAfterExit    bool     `yaml:"remain_after_exit"`
	WantedBy           []string `yaml:"wanted_by"`
}

// Unit represents a database record for a unit.
type Unit struct {
	ID            int64
	Name          string
	Type          string
	CleanupPolicy string
	SHA1Hash      []byte
	UserMode      bool
	RepositoryID  int64
	CreatedAt     string
}
