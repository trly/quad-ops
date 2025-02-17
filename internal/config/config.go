package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Git struct {
		RepoURL string `yaml:"repo_url"`
		Target  string `yaml:"target"`
	} `yaml:"git"`
	Paths struct {
		ManifestsDir string `yaml:"manifests_dir"`
		QuadletDir   string `yaml:"quadlet_dir"`
	} `yaml:"paths"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return &config, err
}
