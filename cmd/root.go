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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/trly/quad-ops/cmd/unit"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/logger"
	"github.com/trly/quad-ops/internal/validation"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootCommand struct{}

var (
	cfg            *config.Config
	userMode       bool
	configFilePath string
	dbPath         string
	quadletDir     string
	repositoryDir  string
	verbose        bool
)

func (c *RootCommand) GetCobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "quad-ops",
		Short: "Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.",
		Long: `Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from Docker Compose files and handles unit reloading and restarting.`,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cfg = config.GetConfig()
			logger.Init(verbose)

			if verbose {
				fmt.Printf("%s using config: %s\n\n", cmd.Root().Use, viper.GetViper().ConfigFileUsed())
				cfg.Verbose = verbose
			}

			if userMode {
				cfg.UserMode = userMode
				cfg.RepositoryDir = os.ExpandEnv("$HOME/.config/quad-ops/repositories")
				cfg.QuadletDir = os.ExpandEnv("$HOME/.config/containers/systemd")
			}

			if repositoryDir != "" {
				cfg.RepositoryDir = repositoryDir
			}

			if quadletDir != "" {
				cfg.QuadletDir = quadletDir
			}

			if dbPath != "" {
				cfg.DBPath = dbPath
			} else {
				cfg.DBPath = filepath.Join(
					filepath.Dir(viper.GetViper().ConfigFileUsed()), "quad-ops.db",
				)
			}

			err := validation.VerifySystemRequirements()
			if err != nil {
				logger.GetLogger().Error("System requirements not met", "err", err)
			}

			err = db.Up(*cfg)
			if err != nil {
				log.Fatalf("failed to initialize database: %v", err)
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&userMode, "user", "u", false, "Run in user mode")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&quadletDir, "quadlet-dir", "", "Path to the quadlet directory")
	rootCmd.PersistentFlags().StringVar(&repositoryDir, "repository-dir", "", "Path to the repository directory")

	rootCmd.AddCommand(
		(&ConfigCommand{}).GetCobraCommand(),
		(&SyncCommand{}).GetCobraCommand(),
		(&unit.Command{}).GetCobraCommand(),
	)

	return rootCmd

}
