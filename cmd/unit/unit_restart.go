package unit

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/unit"
)

type RestartCommand struct{}

func (c *RestartCommand) GetCobraCommand() *cobra.Command {
	unitRestartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			name := args[0]
			if err := unit.RestartUnit(name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}
	return unitRestartCmd
}
