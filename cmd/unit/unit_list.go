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
package unit

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/unit"
)

type ListCommand struct{}

var (
	allowedUnitTypes = []string{"container", "volume", "network", "image", "all"}
)

func (c *ListCommand) GetCobraCommand() *cobra.Command {

	unitListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists units currently managed by quad-ops",
		Run: func(_ *cobra.Command, _ []string) {
			headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
			columnFmt := color.New(color.FgYellow).SprintfFunc()
			tbl := table.New("ID", "Name", "Type", "Unit State", "SHA1", "Cleanup Policy", "Created At")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

			dbConn, err := db.Connect()
			if err != nil {
				log.Fatal(err)
			}
			defer dbConn.Close()

			unitRepo := unit.NewUnitRepository(dbConn)
			findAndDisplayUnits(unitRepo, tbl, unitType)
		},
	}

	unitListCmd.Flags().StringVarP(&unitType, "type", "t", "container", "Type of unit to manage (container, volume, network, image, all)")
	err := unitListCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return allowedUnitTypes, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.Fatal(err)
	}
	unitListCmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		return validateUnitType(unitType)
	}

	return unitListCmd
}

func findAndDisplayUnits(unitRepo unit.UnitRepository, tbl table.Table, unitType string) {
	var units []unit.Unit
	var err error

	switch unitType {
	case "", "all":
		units, err = unitRepo.FindAll()
	default:
		units, err = unitRepo.FindByUnitType(unitType)
	}

	if err != nil {
		log.Fatal(err)
	}

	for _, u := range units {
		systemdUnit := &unit.BaseSystemdUnit{
			Name: u.Name,
			Type: u.Type,
		}
		
		unitStatus, err := systemdUnit.GetStatus()
		if err != nil {
			if config.GetConfig().Verbose {
				log.Printf("error getting unit status: %s", err)
			}
			unitStatus = "UNKNOWN"
		}
		tbl.AddRow(u.ID, u.Name, u.Type, unitStatus, hex.EncodeToString(u.SHA1Hash), u.CleanupPolicy, u.CreatedAt)
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
