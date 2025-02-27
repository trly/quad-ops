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
	"quad-ops/internal/db"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/rodaine/table"
)

// listCmd represents the list command
var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "Lists units currently managed by quad-ops",
		Run: func(cmd *cobra.Command, args []string) {
			headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
			columnFmt := color.New(color.FgYellow).SprintfFunc()
			tbl := table.New("ID", "Name", "Type", "SHA1", "Cleanup Policy", "Created At")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

			dbConn, err := db.Connect(cfg)
			if err != nil {
				log.Fatal(err)
			}
			defer dbConn.Close()

			unitRepo := db.NewUnitRepository(dbConn)
			units, err := unitRepo.List()
			if err != nil {
				log.Fatal(err)
			}
			for _, unit := range units {
				tbl.AddRow(unit.ID, unit.Name, unit.Type, hex.EncodeToString(unit.SHA1Hash), unit.CleanupPolicy, unit.CreatedAt)
			}

			tbl.Print()
		},
	}
)

func init() {
	unitCmd.AddCommand(listCmd)
}
