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
package main

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"gopkg.in/yaml.v3"

	"github.com/trly/quad-ops/internal/config"
)

type Globals struct {
	Config  string            `help:"path to config file" type:"path"`
	Debug   bool              `help:"enable debug mode"`
	Verbose bool              `help:"enable verbose output"`
	AppCfg  *config.AppConfig `kong:"-"` // populated by kong configuration loader
}

type CLI struct {
	Globals

	Sync     SyncCmd     `cmd:"" help:"sync repositories, write systemd unit files, and start services"`
	Update   UpdateCmd   `cmd:"" help:"update quad-ops to the latest version"`
	Validate ValidateCmd `cmd:"" help:"validate compose files for use with quad-ops"`
	Version  VersionCmd  `cmd:"" help:"print version information"`
}

func main() {
	cli := CLI{}

	if len(os.Args) < 2 {
		os.Args = append(os.Args, "--help")
	}

	// Determine config file path based on user mode
	var defaultConfigPath string
	if config.IsUserMode() {
		defaultConfigPath = "~/.config/quad-ops/config.yaml"
	} else {
		defaultConfigPath = "/etc/quad-ops/config.yaml"
	}

	ctx := kong.Parse(&cli,
		kong.Name("quad-ops"),
		kong.Description("GitOps framework for podman containers"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"config_default": defaultConfigPath,
		},
		kong.Configuration(kongyaml.Loader, defaultConfigPath),
	)

	// Load the config file to populate AppConfig
	configPath := cli.Config
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// Expand ~ to home directory
	if configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			ctx.Fatalf("failed to get home directory: %v", err)
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		ctx.Fatalf("failed to read config file %s: %v", configPath, err)
	}

	cfg := &config.AppConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		ctx.Fatalf("failed to parse config file %s: %v", configPath, err)
	}
	cli.AppCfg = cfg

	err = ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
