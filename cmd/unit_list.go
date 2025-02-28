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
	"encoding/hex"
	"log"

	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/db/model"
	"github.com/trly/quad-ops/internal/systemd"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/rodaine/table"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "Lists units currently managed by quad-ops",
		Long: `Usage:

quad-ops unit list

ID  Name           Type       SHA1                                      Cleanup Policy  Created At                    
1   quad-ops-demo  container  d9c614ad03ddbc5e3f8ba2566c40a9aed57e3368  keep            0001-01-01 00:00:00 +0000 UTC 
2   quad-ops-demo  volume     7449585cd761d0a88ba11e72e98470be6627e2b2  keep            0001-01-01 00:00:00 +0000 UTC 
3   quad-ops-demo  network    a65071ee7f0e2b77051f5faf2017e4e08fe3bb3f  keep            0001-01-01 00:00:00 +0000 UTC 
4   quad-ops-demo  image      c68a70c3bb517583a62d7f19c6a0bd0385fed83b  keep            0001-01-01 00:00:00 +0000 UTC`,
		Run: func(cmd *cobra.Command, args []string) {
			headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
			columnFmt := color.New(color.FgYellow).SprintfFunc()
			tbl := table.New("ID", "Name", "Type", "Unit State", "SHA1", "Cleanup Policy", "Created At")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

			dbConn, err := db.Connect(cfg)
			if err != nil {
				log.Fatal(err)
			}
			defer dbConn.Close()

			unitRepo := db.NewUnitRepository(dbConn)
			findAndDisplayUnits(unitRepo, tbl, unitType)
		},
	}
)

func findAndDisplayUnits(unitRepo *db.UnitRepository, tbl table.Table, unitType string) {
	var units []model.Unit
	var err error

	if unitType != "" {
		units, err = unitRepo.FindByUnitType(unitType)
	} else {
		units, err = unitRepo.FindAll()
	}

	if err != nil {
		log.Fatal(err)
	}

	for _, unit := range units {
		unitStatus, err := systemd.GetUnitStatus(*cfg, unit.Name, unit.Type)
		if err != nil {
			if cfg.Verbose {
				log.Printf("error getting unit status: %s", err)
			}
			unitStatus = "UNKNOWN"
		}
		tbl.AddRow(unit.ID, unit.Name, unit.Type, unitStatus, hex.EncodeToString(unit.SHA1Hash), unit.CleanupPolicy, unit.CreatedAt)
	}
	tbl.Print()
}
func init() {
	unitCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&unitType, "type", "t", "", "Type of unit to manage (container, volume, network, image)")
}
