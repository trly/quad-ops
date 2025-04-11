package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

var cfg *Config

// Default configuration values for the quad-ops system.
// These constants define the default values for various configuration
// settings, such as the repository directory, sync interval, quadlet
// directory, database path, user mode, and verbosity.
const (
	DefaultRepositoryDir = "/var/lib/quad-ops"
	DefaultSyncInterval  = 5 * time.Minute
	DefaultQuadletDir    = "/etc/containers/systemd"
	DefaultDBPath        = "/var/lib/quad-ops/quad-ops.db"
	DefaultUserMode      = false
	DefaultVerbose       = false
)

// RepositoryConfig represents a repository that is managed by the quad-ops system.
// It contains information about the repository, including its name, URL, target
// directory, and cleanup policy.
type RepositoryConfig struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url"`
	Reference  string `yaml:"ref,omitempty"`
	ComposeDir string `yaml:"composeDir,omitempty"`
	Cleanup    string `yaml:"cleanup,omitempty"`
}

// Config represents the configuration for the quad-ops system. It contains
// various settings such as the repository directory, sync interval, quadlet
// directory, database path, user mode, and verbosity.
type Config struct {
	RepositoryDir string             `yaml:"repositoryDir"`
	SyncInterval  time.Duration      `yaml:"syncInterval"`
	QuadletDir    string             `yaml:"quadletDir"`
	Repositories  []RepositoryConfig `yaml:"repositories"`
	DBPath        string             `yaml:"dbPath"`
	UserMode      bool               `yaml:"userMode"`
	Verbose       bool               `yaml:"verbose"`
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
	cfg := &Config{
		RepositoryDir: DefaultRepositoryDir,
		SyncInterval:  DefaultSyncInterval,
		QuadletDir:    DefaultQuadletDir,
		DBPath:        DefaultDBPath,
		UserMode:      DefaultUserMode,
		Verbose:       DefaultVerbose,
	}

	viper.SetDefault("repositoryDir", DefaultRepositoryDir)
	viper.SetDefault("syncInterval", DefaultSyncInterval)
	viper.SetDefault("quadletDir", DefaultQuadletDir)
	viper.SetDefault("dbPath", DefaultDBPath)
	viper.SetDefault("userMode", DefaultUserMode)
	viper.SetDefault("verbose", DefaultVerbose)

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
