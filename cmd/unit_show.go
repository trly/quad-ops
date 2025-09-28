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
	"os"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/sorting"
)

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
	unitShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the contents of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Validate system requirements for unit operations
			if err := app.Validator.SystemRequirements(); err != nil {
				app.Logger.Error("System requirements not met", "error", err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			app := c.getApp(cmd)
			name := args[0]

			// Validate unit name to prevent command injection
			if err := sorting.ValidateUnitName(name); err != nil {
				app.Logger.Error("Invalid unit name", "error", err, "name", name)
				os.Exit(1)
			}

			err := app.UnitManager.Show(name, unitType)
			if err != nil {
				app.Logger.Error("Failed to show unit", "error", err)
				os.Exit(1)
			}
		},
	}
	return unitShowCmd
}
