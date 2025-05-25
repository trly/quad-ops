// Package config provides configuration management for quad-ops
package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Provider defines the interface for configuration providers.
type Provider interface {
	// GetConfig returns the current application configuration.
	GetConfig() *Settings
	// SetConfig sets the application configuration.
	SetConfig(c *Settings)
	// InitConfig initializes the application configuration.
	InitConfig() *Settings
	// SetConfigFilePath sets the configuration file path.
	SetConfigFilePath(p string)
}

// defaultConfigProvider implements the Provider interface.
type defaultConfigProvider struct {
	cfg *Settings
}

// NewDefaultConfigProvider creates a new default config provider.
func NewDefaultConfigProvider() Provider {
	return &defaultConfigProvider{}
}

var defaultProvider = NewDefaultConfigProvider()
var cfg *Settings

// Default configuration values for the quad-ops system.
// These constants define the default values for various configuration
// settings, such as the repository directory, sync interval, quadlet
// directory, database path, user mode, and verbosity.
const (
	DefaultRepositoryDir         = "/var/lib/quad-ops"
	DefaultSyncInterval          = 5 * time.Minute
	DefaultQuadletDir            = "/etc/containers/systemd"
	DefaultDBPath                = "/var/lib/quad-ops/quad-ops.db"
	DefaultUserRepositoryDir     = "$HOME/.local/share/quad-ops"
	DefaultUserQuadletDir        = "$HOME/.config/containers/systemd"
	DefaultUserDBPath            = "$HOME/.local/share/quad-ops/quad-ops.db"
	DefaultUserMode              = false
	DefaultVerbose               = false
	DefaultUsePodmanDefaultNames = false
)

// Repository represents a repository that is managed by the quad-ops system.
// It contains information about the repository, including its name, URL, target
// directory, and cleanup policy.
type Repository struct {
	Name                  string `yaml:"name"`
	URL                   string `yaml:"url"`
	Reference             string `yaml:"ref,omitempty"`
	ComposeDir            string `yaml:"composeDir,omitempty"`
	Cleanup               string `yaml:"cleanup,omitempty"`
	UsePodmanDefaultNames bool   `yaml:"usePodmanDefaultNames,omitempty"`
}

// Settings represents the configuration for the quad-ops system. It contains
// various settings such as the repository directory, sync interval, quadlet
// directory, database path, user mode, and verbosity.
type Settings struct {
	RepositoryDir         string        `yaml:"repositoryDir"`
	SyncInterval          time.Duration `yaml:"syncInterval"`
	QuadletDir            string        `yaml:"quadletDir"`
	Repositories          []Repository  `yaml:"repositories"`
	DBPath                string        `yaml:"dbPath"`
	UserMode              bool          `yaml:"userMode"`
	Verbose               bool          `yaml:"verbose"`
	UsePodmanDefaultNames bool          `yaml:"usePodmanDefaultNames"`
}

// Implementation of ConfigProvider methods for defaultConfigProvider

func (p *defaultConfigProvider) SetConfig(c *Settings) {
	p.cfg = c
}

func (p *defaultConfigProvider) GetConfig() *Settings {
	return p.cfg
}

func (p *defaultConfigProvider) SetConfigFilePath(path string) {
	viper.SetConfigFile(path)
}

func (p *defaultConfigProvider) InitConfig() *Settings {
	p.cfg = initConfigInternal()
	return p.cfg
}

// For backward compatibility - pass through to default provider

// SetConfig sets the application configuration.
func SetConfig(c *Settings) {
	defaultProvider.SetConfig(c)
	cfg = c
}

// GetConfig returns the current application configuration.
func GetConfig() *Settings {
	return defaultProvider.GetConfig()
}

// SetConfigFilePath sets the configuration file path.
func SetConfigFilePath(p string) {
	defaultProvider.SetConfigFilePath(p)
}

// InitConfig initializes the application configuration.
func InitConfig() *Settings {
	cfg = defaultProvider.InitConfig()
	return cfg
}

// Internal function to initialize configuration.
func initConfigInternal() *Settings {
	cfg := &Settings{
		RepositoryDir:         DefaultRepositoryDir,
		SyncInterval:          DefaultSyncInterval,
		QuadletDir:            DefaultQuadletDir,
		DBPath:                DefaultDBPath,
		UserMode:              DefaultUserMode,
		Verbose:               DefaultVerbose,
		UsePodmanDefaultNames: DefaultUsePodmanDefaultNames,
	}

	viper.SetDefault("repositoryDir", DefaultRepositoryDir)
	viper.SetDefault("syncInterval", DefaultSyncInterval)
	viper.SetDefault("quadletDir", DefaultQuadletDir)
	viper.SetDefault("dbPath", DefaultDBPath)
	viper.SetDefault("userMode", DefaultUserMode)
	viper.SetDefault("verbose", DefaultVerbose)
	viper.SetDefault("usePodmanDefaultNames", DefaultUsePodmanDefaultNames)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(os.ExpandEnv("$HOME/.config/quad-ops"))
	viper.AddConfigPath("/etc/opt/quad-ops")
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
