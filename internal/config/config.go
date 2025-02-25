package config

import "time"

const (
	Keep   = "keep"
	Delete = "delete"
)

type Repository struct {
	Name    string        `yaml:"name"`
	URL     string        `yaml:"url"`
	Target  string        `yaml:"target"`
	Cleanup CleanupPolicy `yaml:"cleanup"`
}

type Config struct {
	RepositoryDir string        `yaml:"repositoryDir"`
	SyncInterval  time.Duration `yaml:"syncInterval"`
	QuadletDir    string        `yaml:"quadletDir"`
	Repositories  []Repository  `yaml:"repositories"`
	DBPath        string        `yaml:"dbPath"`
	Verbose       bool          `yaml:"verbose"`
}

type CleanupPolicy struct {
	Action string `yaml:"action"`
}
