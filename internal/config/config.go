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

// DefaultProvider returns the default configuration provider instance.
func DefaultProvider() Provider {
	return defaultProvider
}

// Default configuration values for the quad-ops system.
// These constants define the default values for various configuration
// settings, such as the repository directory, sync interval, quadlet
// directory, database path, user mode, and verbosity.
const (
	DefaultRepositoryDir         = "/var/lib/quad-ops"
	DefaultSyncInterval          = 5 * time.Minute
	DefaultQuadletDir            = "/etc/containers/systemd"
	DefaultUserRepositoryDir     = "$HOME/.local/share/quad-ops"
	DefaultUserQuadletDir        = "$HOME/.config/containers/systemd"
	DefaultUserMode              = false
	DefaultVerbose               = false
	DefaultUsePodmanDefaultNames = false
	DefaultUnitStartTimeout      = 10 * time.Second
	DefaultImagePullTimeout      = 30 * time.Second
)

// Repository represents a repository that is managed by the quad-ops system.
// It contains information about the repository, including its name, URL, target
// directory, and compose directory.
type Repository struct {
	Name                  string `yaml:"name"`
	URL                   string `yaml:"url"`
	Reference             string `yaml:"ref,omitempty"`
	ComposeDir            string `yaml:"composeDir,omitempty"`
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
	UserMode              bool          `yaml:"userMode"`
	Verbose               bool          `yaml:"verbose"`
	UsePodmanDefaultNames bool          `yaml:"usePodmanDefaultNames"`
	UnitStartTimeout      time.Duration `yaml:"unitStartTimeout"`
	ImagePullTimeout      time.Duration `yaml:"imagePullTimeout"`
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

// Internal function to initialize configuration.
func initConfigInternal() *Settings {
	cfg := &Settings{
		RepositoryDir:         DefaultRepositoryDir,
		SyncInterval:          DefaultSyncInterval,
		QuadletDir:            DefaultQuadletDir,
		UserMode:              DefaultUserMode,
		Verbose:               DefaultVerbose,
		UsePodmanDefaultNames: DefaultUsePodmanDefaultNames,
		UnitStartTimeout:      DefaultUnitStartTimeout,
		ImagePullTimeout:      DefaultImagePullTimeout,
	}

	viper.SetDefault("repositoryDir", DefaultRepositoryDir)
	viper.SetDefault("syncInterval", DefaultSyncInterval)
	viper.SetDefault("quadletDir", DefaultQuadletDir)
	viper.SetDefault("userMode", DefaultUserMode)
	viper.SetDefault("verbose", DefaultVerbose)
	viper.SetDefault("usePodmanDefaultNames", DefaultUsePodmanDefaultNames)
	viper.SetDefault("unitStartTimeout", DefaultUnitStartTimeout)
	viper.SetDefault("imagePullTimeout", DefaultImagePullTimeout)

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
