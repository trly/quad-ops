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
	"encoding/hex"
	"fmt"
	"os"

	"github.com/SerhiiCho/timeago/v3"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// ListCommand represents the unit list command.
type ListCommand struct{}

var (
	allowedUnitTypes = []string{"container", "volume", "network", "image", "all"}
)

// GetCobraCommand returns the cobra command for listing units.
func (c *ListCommand) GetCobraCommand() *cobra.Command {
	unitListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists units currently managed by quad-ops",
		Run: func(_ *cobra.Command, _ []string) {
			headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
			columnFmt := color.New(color.FgYellow).SprintfFunc()
			tbl := table.New("ID", "Name", "Type", "Unit State", "SHA1", "Updated")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

			unitRepo := repository.NewRepository()
			findAndDisplayUnits(unitRepo, tbl, unitType)
		},
	}

	unitListCmd.Flags().StringVarP(&unitType, "type", "t", "container", "Type of unit to manage (container, volume, network, image, all)")
	err := unitListCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return allowedUnitTypes, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.GetLogger().Error("Error registering flag completion", "error", err)
		os.Exit(1)
	}
	unitListCmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		return validateUnitType(unitType)
	}

	return unitListCmd
}

func findAndDisplayUnits(unitRepo repository.Repository, tbl table.Table, unitType string) {
	var units []repository.Unit
	var err error

	switch unitType {
	case "", "all":
		units, err = unitRepo.FindAll()
	default:
		units, err = unitRepo.FindByUnitType(unitType)
	}

	if err != nil {
		log.GetLogger().Error("Error finding units", "error", err)
		os.Exit(1)
	}

	for _, u := range units {
		systemdUnit := systemd.NewBaseUnit(u.Name, u.Type)

		unitStatus, err := systemdUnit.GetStatus()
		if err != nil {
			log.GetLogger().Debug("Error getting unit status", "error", err)
			unitStatus = "UNKNOWN"
		}
		updateAtString, err := timeago.Parse(u.UpdatedAt)
		if err != nil {
			log.GetLogger().Debug("Error parsing update at time", "error", err)
			updateAtString = "UNKNOWN"
		}
		tbl.AddRow(u.ID, u.Name, u.Type, unitStatus, hex.EncodeToString(u.SHA1Hash), updateAtString)
	}
	tbl.Print()
}

func validateUnitType(unitType string) error {
	for _, allowedType := range allowedUnitTypes {
		if unitType == allowedType {
			return nil
		}
	}
	return fmt.Errorf("invalid unit type: %s, allowed types are: %v", unitType, allowedUnitTypes)
}
