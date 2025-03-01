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
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/systemd"
)

var (
	unitType string

	unitCmd = &cobra.Command{
		Use:   "unit",
		Short: "subcommands for managing and viewing quadlet units",
	}

	startCmd = &cobra.Command{
		Use:   "start [name]",
		Short: "Start a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := systemd.StartUnit(*cfg, name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}

	stopCmd = &cobra.Command{
		Use:   "stop [name]",
		Short: "Stop a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := systemd.StopUnit(*cfg, name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}

	restartCmd = &cobra.Command{
		Use:   "restart [name]",
		Short: "Restart a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if err := systemd.RestartUnit(*cfg, name, unitType); err != nil {
				log.Fatal(err)
			}
		},
	}

	showCmd = &cobra.Command{
		Use:   "show [name]",
		Short: "Show the status of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			status := systemd.ShowUnit(*cfg, name, unitType)
			log.Println(status)
		},
	}
)

func init() {
	rootCmd.AddCommand(unitCmd)
	unitCmd.AddCommand(startCmd, stopCmd, restartCmd, showCmd)
	for _, cmd := range []*cobra.Command{startCmd, stopCmd, restartCmd, showCmd} {
		cmd.Flags().StringVarP(&unitType, "type", "t", "", "Type of unit to manage (container, volume, network, image)")
		cmd.MarkFlagRequired("type")
	}
}
