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

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/validation"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg            *config.Config
	userMode       bool
	configFilePath string
	dbPath         string
	quadletDir     string
	repositoryDir  string
	verbose        bool
	rootCmd        = &cobra.Command{
		Use:   "quad-ops",
		Short: "Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.",
		Long: `Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from YAML manifests and handles unit reloading andd restarting.`,
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func SetConfig(c *config.Config) {
	cfg = c
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&userMode, "user", "u", false, "Run in user mode")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&quadletDir, "quadlet-dir", "", "Path to the quadlet directory")
	rootCmd.PersistentFlags().StringVar(&repositoryDir, "repository-dir", "", "Path to the repository directory")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {

		if verbose {
			fmt.Printf("%s using config: %s\n\n", rootCmd.Use, viper.GetViper().ConfigFileUsed())
			cfg.Verbose = verbose
		}

		if userMode {
			cfg.UserMode = userMode
		}

		if repositoryDir != "" {
			cfg.RepositoryDir = repositoryDir
		} else if cfg.RepositoryDir == "" && cfg.UserMode {
			cfg.RepositoryDir = os.ExpandEnv("$HOME/.config/quad-ops/repositories")
		} else if cfg.RepositoryDir == "" && !cfg.UserMode {
			cfg.RepositoryDir = "/etc/quad-ops/repositories"
		}

		if quadletDir != "" {
			cfg.QuadletDir = quadletDir
		} else if cfg.QuadletDir == "" && cfg.UserMode {
			cfg.QuadletDir = os.ExpandEnv("$HOME/.config/containers/systemd")
		} else if cfg.QuadletDir == "" && !cfg.UserMode {
			cfg.QuadletDir = "/etc/containers/systemd"
		}

		if dbPath != "" {
			cfg.DBPath = dbPath
		} else {
			cfg.DBPath = filepath.Join(
				filepath.Dir(viper.GetViper().ConfigFileUsed()), "quad-ops.db",
			)
		}

		validation.VerifySystemRequirements(*cfg)

		err := db.Up(*cfg)
		if err != nil {
			log.Fatalf("failed to initialize database: %v", err)
			os.Exit(1)
		}
	}
}

func GetConfigFilePath() string {
	return configFilePath
}
