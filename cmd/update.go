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
	"fmt"

	"github.com/sanbornm/go-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

// UpdateCommand represents the update command.
type UpdateCommand struct{}

// GetCobraCommand returns the cobra command for updating the binary.
func (c *UpdateCommand) GetCobraCommand() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update quad-ops to the latest version",
		Long:  `Update quad-ops to the latest version from GitHub releases.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			updater := &selfupdate.Updater{
				CurrentVersion: Version,
				ApiURL:         "https://api.github.com/repos/trly/quad-ops/",
				BinURL:         "https://github.com/trly/quad-ops/releases/download/",
				DiffURL:        "https://github.com/trly/quad-ops/releases/download/",
				Dir:            "update/",
				CmdName:        "quad-ops",
			}

			fmt.Printf("Current version: %s\n", Version)
			fmt.Println("Checking for updates...")

			latestVersion, err := updater.UpdateAvailable()
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			if latestVersion != "" {
				fmt.Printf("Update available! New version: %s\n", latestVersion)
				fmt.Println("Downloading...")
				err := updater.BackgroundRun()
				if err != nil {
					return fmt.Errorf("failed to update: %w", err)
				}
				fmt.Println("Update completed successfully!")
			} else {
				fmt.Println("You are already running the latest version.")
			}

			return nil
		},
	}

	return updateCmd
}
