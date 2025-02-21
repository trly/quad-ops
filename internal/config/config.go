package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type HostConfig struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern"`
}

type Repository struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Target string `yaml:"target"`
}

type Config struct {
	RepositoryDir string       `yaml:"repositoryDir"`
	QuadletDir    string       `yaml:"quadletDir"`
	Repositories  []Repository `yaml:"repositories"`
	DBPath        string       `yaml:"dbPath"`
}

func LoadConfig(path string, userMode bool, verbose bool) (*Config, error) {
	var cfg Config

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		if err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
		}
	}

	if cfg.QuadletDir == "" {
		if userMode {
			cfg.QuadletDir = os.ExpandEnv("${HOME}/.config/containers/systemd")
		} else {
			cfg.QuadletDir = "/etc/containers/systemd"
		}
	}

	if cfg.RepositoryDir == "" {
		if userMode {
			cfg.RepositoryDir = os.ExpandEnv("${HOME}/.config/quad-ops/manifests")
		} else {
			cfg.RepositoryDir = "/opt/quad-ops/repositories"
		}
	}

	if cfg.DBPath == "" {
		if userMode {
			cfg.DBPath = os.ExpandEnv("${HOME}/.config/quad-ops/quad-ops.db")
		} else {
			cfg.DBPath = os.ExpandEnv("/opt/quad-ops/quad-ops.db")
		}
	}

	if verbose {
		yamlData, _ := yaml.Marshal(cfg)
		log.Printf("Loaded config:\n%s", string(yamlData))
	}

	return &cfg, nil
}
