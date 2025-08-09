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
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/sorting"
	"github.com/trly/quad-ops/internal/systemd"
)

// ShowCommand represents the unit show command.
type ShowCommand struct{}

// GetCobraCommand returns the cobra command for showing unit details.
func (c *ShowCommand) GetCobraCommand() *cobra.Command {
	unitShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the contents of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			name := args[0]

			// Validate unit name to prevent command injection
			if err := sorting.ValidateUnitName(name); err != nil {
				log.GetLogger().Error("Invalid unit name", "error", err, "name", name)
				os.Exit(1)
			}

			systemdUnit := systemd.NewBaseUnit(name, unitType)

			err := systemdUnit.Show()
			if err != nil {
				log.GetLogger().Error("Failed to show unit", "error", err)
				os.Exit(1)
			}
		},
	}
	return unitShowCmd
}
