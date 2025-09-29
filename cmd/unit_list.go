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
	"context"
	"encoding/hex"
	"fmt"

	"github.com/SerhiiCho/timeago/v3"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/repository"
)

// ListOptions holds list command options.
type ListOptions struct {
	UnitType string
}

// ListDeps holds list dependencies.
type ListDeps struct {
	CommonDeps
}

// ListCommand represents the unit list command.
type ListCommand struct{}

// NewListCommand creates a new ListCommand.
func NewListCommand() *ListCommand {
	return &ListCommand{}
}

// getApp retrieves the App from the command context.
func (c *ListCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

var (
	allowedUnitTypes = []string{"container", "volume", "network", "image", "all"}
)

// GetCobraCommand returns the cobra command for listing units.
func (c *ListCommand) GetCobraCommand() *cobra.Command {
	var opts ListOptions

	unitListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists units currently managed by quad-ops",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			if err := app.Validator.SystemRequirements(); err != nil {
				return err
			}
			return validateUnitType(opts.UnitType)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	unitListCmd.Flags().StringVarP(&opts.UnitType, "type", "t", "container", "Type of unit to manage (container, volume, network, image, all)")
	err := unitListCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return allowedUnitTypes, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		return unitListCmd
	}

	return unitListCmd
}

// Run executes the list command with injected dependencies.
func (c *ListCommand) Run(_ context.Context, app *App, opts ListOptions, deps ListDeps) error {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()
	tbl := table.New("ID", "Name", "Type", "Unit State", "SHA1", "Updated")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	return c.findAndDisplayUnits(app, tbl, opts.UnitType, deps)
}

// buildDeps creates production dependencies for the list command.
func (c *ListCommand) buildDeps(app *App) ListDeps {
	return ListDeps{
		CommonDeps: NewRootDeps(app),
	}
}

func (c *ListCommand) findAndDisplayUnits(app *App, tbl table.Table, unitType string, deps ListDeps) error {
	var units []repository.Unit
	var err error

	switch unitType {
	case "", "all":
		units, err = app.UnitRepo.FindAll()
	default:
		units, err = app.UnitRepo.FindByUnitType(unitType)
	}

	if err != nil {
		return fmt.Errorf("error finding units: %w", err)
	}

	for _, u := range units {
		unitStatus, err := app.UnitManager.GetStatus(u.Name, u.Type)
		if err != nil {
			deps.Logger.Debug("Error getting unit status", "error", err)
			unitStatus = "UNKNOWN"
		}
		updateAtString, err := timeago.Parse(u.UpdatedAt)
		if err != nil {
			deps.Logger.Debug("Error parsing update at time", "error", err)
			updateAtString = "UNKNOWN"
		}
		tbl.AddRow(u.ID, u.Name, u.Type, unitStatus, hex.EncodeToString(u.SHA1Hash), updateAtString)
	}
	tbl.Print()
	return nil
}

func validateUnitType(unitType string) error {
	// Allow empty string as it defaults to container behavior
	if unitType == "" {
		return nil
	}
	for _, allowedType := range allowedUnitTypes {
		if unitType == allowedType {
			return nil
		}
	}
	return fmt.Errorf("invalid unit type: %s, allowed types are: %v", unitType, allowedUnitTypes)
}
