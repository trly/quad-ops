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

// GetCobraCommand returns the cobra command for stopping all managed units.
func (c *DownCommand) GetCobraCommand() *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop all managed units",
		Long:  "Stop all managed units synchronized from repositories.",
		Run: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Get all units
			units, err := app.UnitRepo.FindAll()
			if err != nil {
				app.Logger.Error("Failed to get units from database", "error", err)
				os.Exit(1)
			}

			if len(units) == 0 {
				fmt.Println("No managed units found")
				return
			}

			fmt.Printf("Stopping %d managed units...\n", len(units))

			successCount := 0
			failCount := 0

			// Stop each unit
			for _, u := range units {
				err := app.UnitManager.Stop(u.Name, u.Type)
				if err != nil {
					app.Logger.Error("Failed to stop unit", "name", u.Name, "error", err)
					failCount++
				} else {
					successCount++
				}
			}

			fmt.Printf("Successfully stopped %d units, failed to stop %d units\n", successCount, failCount)
		},
	}

	return downCmd
}
