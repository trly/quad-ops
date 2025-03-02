package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

var cfg *Config
var configFilePath string

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
	UserMode      bool          `yaml:"userMode"`
	Verbose       bool          `yaml:"verbose"`
}

type CleanupPolicy struct {
	Action string `yaml:"action"`
}

func SetConfig(c *Config) {
	cfg = c
}

func GetConfig() *Config {
	return cfg
}

func SetConfigFilePath(p string) {
	viper.SetConfigFile(p)
}

func InitConfig() *Config {
	cfg := &Config{}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(os.ExpandEnv("$HOME/.config/quad-ops"))
	viper.AddConfigPath("/etc/quad-ops")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	if err := viper.Unmarshal(cfg); err != nil {
		panic(err)
	}

	return cfg
}
