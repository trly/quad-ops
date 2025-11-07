// Package cmd provides config show command functionality for quad-ops CLI
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/trly/quad-ops/internal/config"
)

// ConfigShowCommand represents the config show command.
type ConfigShowCommand struct{}

// NewConfigShowCommand creates a new ConfigShowCommand.
func NewConfigShowCommand() *ConfigShowCommand {
	return &ConfigShowCommand{}
}

// getApp retrieves the App from the command context.
func (c *ConfigShowCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for config show operations.
func (c *ConfigShowCommand) GetCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		Long:  "Display the current configuration including defaults and overrides",
		Run: func(cmd *cobra.Command, _ []string) {
			// Try to get config from app context first, fall back to global cfg
			var config *config.Settings
			if app := c.getApp(cmd); app != nil {
				config = app.Config
			} else {
				config = cfg
			}

			output, err := yaml.Marshal(config)
			if err != nil {
				fmt.Printf("Error marshalling config: %v\n", err)
				return
			}
			fmt.Println(string(output))
		},
	}
}
