package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/systemd"
)

type UnitRestartCommand struct{}

func (c *UnitRestartCommand) GetCobraCommand() *cobra.Command {
	unitRestartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := systemd.RestartUnit(*cfg, name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}
	return unitRestartCmd
}
