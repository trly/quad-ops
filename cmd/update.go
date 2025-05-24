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

	"github.com/creativeprojects/go-selfupdate"
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
			fmt.Printf("Current version: %s\n", Version)
			fmt.Println("Checking for updates...")

			// Detect latest version
			latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("trly/quad-ops"))
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			if !found {
				fmt.Println("No release found")
				return nil
			}

			if latest.LessOrEqual(Version) {
				fmt.Println("You are already running the latest version.")
				return nil
			}

			fmt.Printf("Update available! New version: %s\n", latest.Version())
			fmt.Println("Downloading and applying update...")

			// Get current executable path
			exe, err := selfupdate.ExecutablePath()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			// Update to the latest version
			if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
				return fmt.Errorf("failed to update: %w", err)
			}

			fmt.Printf("Update completed successfully! Please restart %s to use the new version.\n", "quad-ops")
			return nil
		},
	}

	return updateCmd
}
