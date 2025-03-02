package unit

type VolumeConfig struct {
	Label    []string `yaml:"label"`
	Device   string   `yaml:"device"`
	Options  []string `yaml:"options"`
	UID      int      `yaml:"uid"`
	GID      int      `yaml:"gid"`
	Mode     string   `yaml:"mode"`
	Chown    bool     `yaml:"chown"`
	Selinux  bool     `yaml:"selinux"`
	Copy     bool     `yaml:"copy"`
	Group    string   `yaml:"group"`
	Size     string   `yaml:"size"`
	Capacity string   `yaml:"capacity"`
	Type     string   `yaml:"type"`
}
