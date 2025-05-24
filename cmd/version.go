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
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

// Build information set by goreleaser.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// VersionCommand represents the version command.
type VersionCommand struct{}

// GetCobraCommand returns the cobra command for displaying version information.
func (c *VersionCommand) GetCobraCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Show version information for quad-ops.`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("quad-ops version %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built: %s\n", Date)
			fmt.Printf("  go: %s\n", runtime.Version())

			// Check for updates
			c.checkForUpdates()
		},
	}

	return versionCmd
}

// checkForUpdates checks if a newer version is available and prints a message if so.
func (c *VersionCommand) checkForUpdates() {
	// Skip update check for development builds
	if Version == "dev" {
		fmt.Println("\nSkipping update check for development build.")
		return
	}

	fmt.Println("\nChecking for updates...")

	// Detect latest version
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("trly/quad-ops"))
	if err != nil {
		fmt.Printf("Failed to check for updates: %v\n", err)
		return
	}

	if !found {
		fmt.Println("No release found")
		return
	}

	if latest.LessOrEqual(Version) {
		fmt.Println("You are running the latest version.")
		return
	}

	fmt.Printf("🚀 Update available! New version: %s\n", latest.Version())
	fmt.Println("Run 'quad-ops update' to update to the latest version.")
}
