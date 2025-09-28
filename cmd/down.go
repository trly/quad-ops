// Package cmd provides the command line interface for quad-ops
/*
Copyright u00a9 2025 Travis Lyons travis.lyons@gmail.com

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

// DownCommand represents the down command for quad-ops CLI.
type DownCommand struct{}

// NewDownCommand creates a new DownCommand.
func NewDownCommand() *DownCommand {
	return &DownCommand{}
}

// getApp retrieves the App from the command context.
func (c *DownCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for stopping managed units.
func (c *DownCommand) GetCobraCommand() *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down [unit-name...]",
		Short: "Stop managed units",
		Long: `Stop managed units synchronized from repositories.

If no unit names are provided, stops all managed units.
If unit names are provided, stops only the specified units.

Examples:
  quad-ops down                    # Stop all units
  quad-ops down web-service        # Stop specific unit
  quad-ops down web api database   # Stop multiple units`,
		PreRun: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Validate system requirements for stopping units
			if err := app.Validator.SystemRequirements(); err != nil {
				app.Logger.Error("System requirements not met", "error", err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			app := c.getApp(cmd)

			// Get units to stop (all units or specified units)
			var unitsToStop []string
			if len(args) == 0 {
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

				for _, u := range units {
					unitsToStop = append(unitsToStop, u.Name)
				}
			} else {
				// Use specified unit names
				unitsToStop = args
			}

			if app.Config.Verbose {
				fmt.Printf("Stopping %d units...\n", len(unitsToStop))
			}

			successCount := 0
			failCount := 0

			// Stop each unit
			for _, unitName := range unitsToStop {
				err := app.UnitManager.Stop(unitName, "container")
				if err != nil {
					app.Logger.Error("Failed to stop unit", "name", unitName, "error", err)
					failCount++
				} else {
					successCount++
				}
			}

			// Only output summary on failure or in verbose mode
			if failCount > 0 {
				fmt.Printf("Failed to stop %d units\n", failCount)
				os.Exit(1)
			} else if app.Config.Verbose {
				fmt.Printf("Successfully stopped %d units\n", successCount)
			}
		},
	}

	return downCmd
}
