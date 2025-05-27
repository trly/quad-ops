package unit

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/systemd"
)

// StatusCommand represents the unit status command.
type StatusCommand struct{}

// GetCobraCommand returns the cobra command for checking unit status.
func (c *StatusCommand) GetCobraCommand() *cobra.Command {
	unitStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			name := args[0]

			// Create a base systemd unit with the provided name and type
			systemdUnit := systemd.NewBaseUnit(name, unitType)

			err := systemdUnit.Show()
			if err != nil {
				fmt.Printf("Error showing unit status: %v\n", err)
			}
		},
	}
	return unitStatusCmd
}
