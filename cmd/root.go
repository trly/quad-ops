// Package cmd provides the command line interface for quad-ops
/*
Copyright © 2025 Travis Lyons travis.lyons@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/sorting"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// contextKey is the type used for context keys to avoid collisions.
type contextKey string

// appContextKey is the context key for the App instance.
const appContextKey = contextKey("app")

// RootCommand represents the root command for quad-ops CLI.
type RootCommand struct{}

var (
	cfg            *config.Settings
	userMode       bool
	configFilePath string
	quadletDir     string
	repositoryDir  string
	verbose        bool
	outputFormat   string
)

// GetCobraCommand returns the cobra root command for quad-ops CLI.
func (c *RootCommand) GetCobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "quad-ops",
		Short: "Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.",
		Long: `Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from Docker Compose files and handles unit reloading and restarting.`,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			configProv := config.NewConfigProvider()
			cfg = configProv.GetConfig()
			logger := log.NewLogger(verbose)

			if verbose {
				fmt.Printf("%s using config: %s\n\n", cmd.Root().Use, viper.GetViper().ConfigFileUsed())
				cfg.Verbose = verbose
			}

			if userMode {
				cfg.UserMode = userMode
				cfg.RepositoryDir = os.ExpandEnv(config.DefaultUserRepositoryDir)
				cfg.QuadletDir = os.ExpandEnv(config.DefaultUserQuadletDir)
			}

			if repositoryDir != "" {
				// Validate repository directory path
				if err := sorting.ValidatePath(repositoryDir); err != nil {
					logger.Error("Invalid repository directory", "path", repositoryDir, "error", err)
					os.Exit(1)
				}
				cfg.RepositoryDir = repositoryDir
			}

			if quadletDir != "" {
				// Validate quadlet directory path
				if err := sorting.ValidatePath(quadletDir); err != nil {
					logger.Error("Invalid quadlet directory", "path", quadletDir, "error", err)
					os.Exit(1)
				}
				cfg.QuadletDir = quadletDir
			}

			// Initialize app and store in context for commands that need it
			app := NewApp(logger, configProv)
			cmd.SetContext(context.WithValue(cmd.Context(), appContextKey, app))
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&userMode, "user", "u", false, "Run in user mode")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&quadletDir, "quadlet-dir", "", "Path to the quadlet directory")
	rootCmd.PersistentFlags().StringVar(&repositoryDir, "repository-dir", "", "Path to the repository directory")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	rootCmd.AddCommand(
		NewConfigCommand().GetCobraCommand(),
		NewSyncCommand().GetCobraCommand(),
		NewDaemonCommand().GetCobraCommand(),
		NewDoctorCommand().GetCobraCommand(),
		NewUnitCommand().GetCobraCommand(),
		NewUpCommand().GetCobraCommand(),
		NewImageCommand().GetCobraCommand(),
		NewDownCommand().GetCobraCommand(),
		NewUpdateCommand().GetCobraCommand(),
		NewValidateCommand().GetCobraCommand(),
		NewVersionCommand().GetCobraCommand(),
	)

	return rootCmd
}
