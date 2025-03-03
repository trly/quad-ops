package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/systemd"
)

type UnitStopCommand struct{}

func (c *UnitStopCommand) GetCobraCommand() *cobra.Command {
	unitStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := systemd.StopUnit(*cfg, name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}
	return unitStopCmd
}
