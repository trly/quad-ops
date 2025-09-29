// Package cmd provides unit command functionality for quad-ops CLI
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/sorting"
)

// StatusOptions holds status command options.
type StatusOptions struct {
	UnitType string
}

// StatusDeps holds status dependencies.
type StatusDeps struct {
	CommonDeps
}

// StatusCommand represents the unit status command.
type StatusCommand struct{}

// NewStatusCommand creates a new StatusCommand.
func NewStatusCommand() *StatusCommand {
	return &StatusCommand{}
}

// getApp retrieves the App from the command context.
func (c *StatusCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for checking unit status.
func (c *StatusCommand) GetCobraCommand() *cobra.Command {
	var opts StatusOptions

	unitStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of a quadlet unit",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps, args[0])
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	unitStatusCmd.Flags().StringVarP(&opts.UnitType, "type", "t", "container", "Type of unit (container, volume, network, image)")

	return unitStatusCmd
}

// Run executes the status command with injected dependencies.
func (c *StatusCommand) Run(_ context.Context, app *App, opts StatusOptions, _ StatusDeps, unitName string) error {
	// Validate unit name to prevent command injection
	if err := sorting.ValidateUnitName(unitName); err != nil {
		return fmt.Errorf("invalid unit name %q: %w", unitName, err)
	}

	err := app.UnitManager.Show(unitName, opts.UnitType)
	if err != nil {
		return fmt.Errorf("error showing unit status for %q: %w", unitName, err)
	}

	return nil
}

// buildDeps creates production dependencies for the status command.
func (c *StatusCommand) buildDeps(app *App) StatusDeps {
	return StatusDeps{
		CommonDeps: NewCommonDeps(app.Logger),
	}
}
