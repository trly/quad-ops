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

// RootOptions holds root command options.
type RootOptions struct {
	UserMode       bool
	ConfigFilePath string
	QuadletDir     string
	RepositoryDir  string
	Verbose        bool
	OutputFormat   string
}

// RootDeps holds root dependencies.
type RootDeps struct {
	CommonDeps
	ValidatePath func(string) error
	ExpandEnv    func(string) string
}

// RootCommand represents the root command for quad-ops CLI.
type RootCommand struct{}

var (
	cfg *config.Settings
)

// GetCobraCommand returns the cobra root command for quad-ops CLI.
func (c *RootCommand) GetCobraCommand() *cobra.Command {
	var opts RootOptions

	rootCmd := &cobra.Command{
		Use:   "quad-ops",
		Short: "Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.",
		Long: `Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from Docker Compose files and handles unit reloading and restarting.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			deps := c.buildDeps()
			return c.persistentPreRun(cmd, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVarP(&opts.UserMode, "user", "u", false, "Run in user mode")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&opts.ConfigFilePath, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&opts.QuadletDir, "quadlet-dir", "", "Path to the quadlet directory")
	rootCmd.PersistentFlags().StringVar(&opts.RepositoryDir, "repository-dir", "", "Path to the repository directory")
	rootCmd.PersistentFlags().StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

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

// buildDeps creates production dependencies for root.
func (c *RootCommand) buildDeps() RootDeps {
	return RootDeps{
		CommonDeps:   CommonDeps{}, // Will be initialized in persistentPreRun
		ValidatePath: sorting.ValidatePath,
		ExpandEnv:    os.ExpandEnv,
	}
}

// persistentPreRun executes the persistent pre-run logic with injected dependencies.
func (c *RootCommand) persistentPreRun(cmd *cobra.Command, opts RootOptions, deps RootDeps) error {
	configProv := config.NewConfigProvider()
	cfg = configProv.GetConfig()
	logger := log.NewLogger(opts.Verbose)

	if opts.Verbose {
		fmt.Printf("%s using config: %s\n\n", cmd.Root().Use, viper.GetViper().ConfigFileUsed())
		cfg.Verbose = opts.Verbose
	}

	if opts.UserMode {
		cfg.UserMode = opts.UserMode
		cfg.RepositoryDir = deps.ExpandEnv(config.DefaultUserRepositoryDir)
		cfg.QuadletDir = deps.ExpandEnv(config.DefaultUserQuadletDir)
	}

	if opts.RepositoryDir != "" {
		// Validate repository directory path
		if err := deps.ValidatePath(opts.RepositoryDir); err != nil {
			return fmt.Errorf("invalid repository directory %s: %w", opts.RepositoryDir, err)
		}
		cfg.RepositoryDir = opts.RepositoryDir
	}

	if opts.QuadletDir != "" {
		// Validate quadlet directory path
		if err := deps.ValidatePath(opts.QuadletDir); err != nil {
			return fmt.Errorf("invalid quadlet directory %s: %w", opts.QuadletDir, err)
		}
		cfg.QuadletDir = opts.QuadletDir
	}

	// Initialize app and store in context for commands that need it
	app := NewApp(logger, configProv)
	app.OutputFormat = opts.OutputFormat
	cmd.SetContext(context.WithValue(cmd.Context(), appContextKey, app))
	return nil
}
