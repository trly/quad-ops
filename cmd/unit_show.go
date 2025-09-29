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

// Package cmd provides unit command functionality for quad-ops CLI
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/sorting"
)

// ShowOptions holds show command options.
type ShowOptions struct {
	UnitType string
}

// ShowDeps holds show dependencies.
type ShowDeps struct {
	CommonDeps
}

// ShowCommand represents the unit show command.
type ShowCommand struct{}

// NewShowCommand creates a new ShowCommand.
func NewShowCommand() *ShowCommand {
	return &ShowCommand{}
}

// getApp retrieves the App from the command context.
func (c *ShowCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for showing unit details.
func (c *ShowCommand) GetCobraCommand() *cobra.Command {
	var opts ShowOptions

	unitShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the contents of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps, args[0])
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	unitShowCmd.Flags().StringVarP(&opts.UnitType, "type", "t", "container", "Type of unit (container, volume, network, image)")

	return unitShowCmd
}

// Run executes the show command with injected dependencies.
func (c *ShowCommand) Run(_ context.Context, app *App, opts ShowOptions, _ ShowDeps, unitName string) error {
	// Validate unit name to prevent command injection
	if err := sorting.ValidateUnitName(unitName); err != nil {
		return fmt.Errorf("invalid unit name %q: %w", unitName, err)
	}

	err := app.UnitManager.Show(unitName, opts.UnitType)
	if err != nil {
		return fmt.Errorf("failed to show unit %q: %w", unitName, err)
	}

	return nil
}

// buildDeps creates production dependencies for the show command.
func (c *ShowCommand) buildDeps(app *App) ShowDeps {
	return ShowDeps{
		CommonDeps: NewRootDeps(app),
	}
}
