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
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// DownCommand represents the down command for quad-ops CLI.
type DownCommand struct{}

// GetCobraCommand returns the cobra command for stopping all managed units.
func (c *DownCommand) GetCobraCommand() *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop all managed units",
		Long:  "Stop all managed units synchronized from repositories.",
		Run: func(_ *cobra.Command, _ []string) {
			// Get all units
			unitRepo := repository.NewRepository()
			units, err := unitRepo.FindAll()
			if err != nil {
				log.GetLogger().Error("Failed to get units from database", "error", err)
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
				systemdUnit := systemd.NewBaseUnit(u.Name, u.Type)
				err := systemdUnit.Stop()
				if err != nil {
					log.GetLogger().Error("Failed to stop unit", "name", u.Name, "error", err)
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
