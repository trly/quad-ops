// Package cmd provides config init command functionality for quad-ops CLI
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/trly/quad-ops/internal/config"
)

// InitOptions holds init command options.
type InitOptions struct {
	Force bool
}

// InitDeps holds init dependencies.
type InitDeps struct {
	CommonDeps
	UserHomeDir func() (string, error)
	MkdirAll    func(string, os.FileMode) error
	WriteFile   func(string, []byte, os.FileMode) error
}

// InitCommand represents the config init command.
type InitCommand struct{}

// NewInitCommand creates a new InitCommand.
func NewInitCommand() *InitCommand {
	return &InitCommand{}
}

// getApp retrieves the App from the command context.
func (c *InitCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for config init.
func (c *InitCommand) GetCobraCommand() *cobra.Command {
	var opts InitOptions

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a default configuration file",
		Long:  "Create a default configuration file in the user configuration directory with example repositories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps()
			return c.Run(app, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	initCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing configuration file")

	return initCmd
}

// Run executes the init command with injected dependencies.
func (c *InitCommand) Run(_ *App, opts InitOptions, deps InitDeps) error {
	// Get user home directory
	homeDir, err := deps.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Determine config directory
	configDir := filepath.Join(homeDir, ".config", "quad-ops")
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil && !opts.Force {
		return fmt.Errorf("configuration file already exists at %s, use --force to overwrite", configFile)
	}

	// Create config directory
	if err := deps.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Create default config
	defaultConfig := &config.Settings{
		Repositories: []config.Repository{
			{
				Name:       "quad-ops-examples",
				URL:        "https://github.com/trly/quad-ops.git",
				Reference:  "main",
				ComposeDir: "examples",
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := deps.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configFile, err)
	}

	fmt.Printf("Configuration file created at %s\n", configFile)
	return nil
}

// buildDeps creates production dependencies for the init command.
func (c *InitCommand) buildDeps() InitDeps {
	return InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: os.UserHomeDir,
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}
}
