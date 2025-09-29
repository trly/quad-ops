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

	"github.com/spf13/cobra"
)

// UpOptions holds up command options.
type UpOptions struct {
	// No specific flags for up command currently
}

// UpDeps holds up dependencies.
type UpDeps struct {
	CommonDeps
}

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

// GetCobraCommand returns the cobra command for starting managed units.
func (c *UpCommand) GetCobraCommand() *cobra.Command {
	var opts UpOptions

	upCmd := &cobra.Command{
		Use:   "up [unit-name...]",
		Short: "Start managed units",
		Long: `Start managed units synchronized from repositories.

If no unit names are provided, starts all managed units.
If unit names are provided, starts only the specified units.

Examples:
  quad-ops up                    # Start all units
  quad-ops up web-service        # Start specific unit
  quad-ops up web api database   # Start multiple units`,
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

	return upCmd
}

// buildDeps creates production dependencies for the up command.
func (c *UpCommand) buildDeps(app *App) UpDeps {
	return UpDeps{
		CommonDeps: NewCommonDeps(app.Logger),
	}
}

// Run executes the up command with injected dependencies.
func (c *UpCommand) Run(_ context.Context, app *App, _ UpOptions, deps UpDeps, args []string) error {
	// Get units to start (all units or specified units)
	var unitsToStart []string
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
			unitsToStart = append(unitsToStart, u.Name)
		}
	} else {
		// Use specified unit names
		unitsToStart = args
	}

	if app.Config.Verbose {
		fmt.Printf("Starting %d units...\n", len(unitsToStart))
	}

	successCount := 0
	failCount := 0

	// Start each unit
	for _, unitName := range unitsToStart {
		// Reset any failed units before attempting to start
		_ = app.UnitManager.ResetFailed(unitName, "container")

		err := app.UnitManager.Start(unitName, "container")
		if err != nil {
			deps.Logger.Error("Failed to start unit", "name", unitName, "error", err)
			failCount++
		} else {
			successCount++
		}
	}

	// Only output summary on failure or in verbose mode
	if failCount > 0 {
		return fmt.Errorf("failed to start %d units", failCount)
	} else if app.Config.Verbose {
		fmt.Printf("Successfully started %d units\n", successCount)
	}

	return nil
}
