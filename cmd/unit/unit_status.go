package unit

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/unit"
)

type UnitStatusCommand struct{}

func (c *UnitStatusCommand) GetCobraCommand() *cobra.Command {
	unitStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			status := unit.ShowUnit(name, unitType)
			log.Println(status)
		},
	}
	return unitStatusCmd
}
