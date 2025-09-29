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
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// DownOptions holds down command options.
type DownOptions struct {
	// No specific flags for down command currently
}

// DownDeps holds down dependencies.
type DownDeps struct {
	CommonDeps
}

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
	var opts DownOptions

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
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return downCmd
}

// buildDeps creates production dependencies for the down command.
func (c *DownCommand) buildDeps(app *App) DownDeps {
	return DownDeps{
		CommonDeps: NewRootDeps(app),
	}
}

// Run executes the down command with injected dependencies.
func (c *DownCommand) Run(_ context.Context, app *App, _ DownOptions, deps DownDeps, args []string) error {
	// Get units to stop (all units or specified units)
	var unitsToStop []string
	if len(args) == 0 {
		// Get all units
		units, err := app.UnitRepo.FindAll()
		if err != nil {
			return fmt.Errorf("failed to get units from database: %w", err)
		}

		if len(units) == 0 {
			if app.Config.Verbose {
				fmt.Println("No managed units found")
			}
			return nil
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
			deps.Logger.Error("Failed to stop unit", "name", unitName, "error", err)
			failCount++
		} else {
			successCount++
		}
	}

	// Only output summary on failure or in verbose mode
	if failCount > 0 {
		return fmt.Errorf("failed to stop %d units", failCount)
	} else if app.Config.Verbose {
		fmt.Printf("Successfully stopped %d units\n", successCount)
	}

	return nil
}
