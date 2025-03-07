package unit

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/unit"
)

type StopCommand struct{}

func (c *StopCommand) GetCobraCommand() *cobra.Command {
	unitStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			name := args[0]
			if err := unit.StopUnit(name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}
	return unitStopCmd
}
