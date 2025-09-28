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
	"os"

	"github.com/spf13/cobra"
)

// UpCommand represents the up command for quad-ops CLI.
type UpCommand struct{}

// NewUpCommand creates a new UpCommand.
func NewUpCommand() *UpCommand {
	return &UpCommand{}
}

// getApp retrieves the App from the command context.
func (c *UpCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for starting all managed units.
func (c *UpCommand) GetCobraCommand() *cobra.Command {
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Start all managed units",
		Long:  "Start all managed units synchronized from repositories.",
		Run: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Get all units
			units, err := app.UnitRepo.FindAll()
			if err != nil {
				app.Logger.Error("Failed to get units from database", "error", err)
				os.Exit(1)
			}

			if len(units) == 0 {
				if app.Config.Verbose {
					fmt.Println("No managed units found")
				}
				return
			}

			if app.Config.Verbose {
				fmt.Printf("Starting %d managed units...\n", len(units))
			}

			successCount := 0
			failCount := 0

			// Start each unit
			for _, u := range units {
				// Reset any failed units before attempting to start
				_ = app.UnitManager.ResetFailed(u.Name, u.Type)

				err := app.UnitManager.Start(u.Name, u.Type)
				if err != nil {
					app.Logger.Error("Failed to start unit", "name", u.Name, "error", err)
					failCount++
				} else {
					successCount++
				}
			}

			// Only output summary on failure or in verbose mode
			if failCount > 0 {
				fmt.Printf("Failed to start %d units\n", failCount)
				os.Exit(1)
			} else if app.Config.Verbose {
				fmt.Printf("Successfully started %d units\n", successCount)
			}
		},
	}

	return upCmd
}
