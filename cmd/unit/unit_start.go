package unit

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/unit"
)

type UnitStartCommand struct{}

func (c *UnitStartCommand) GetCobraCommand() *cobra.Command {
	unitStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := unit.StartUnit(name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}
	return unitStartCmd
}
