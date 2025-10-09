// Package cmd provides unit command functionality for quad-ops CLI
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/sorting"
)

// StatusOptions holds status command options.
type StatusOptions struct{}

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
		Use:   "status SERVICE",
		Short: "Show the status of a service",
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

	return unitStatusCmd
}

// Run executes the status command with injected dependencies.
func (c *StatusCommand) Run(ctx context.Context, app *App, _ StatusOptions, deps StatusDeps, serviceName string) error {
	// Validate service name to prevent command injection
	if err := sorting.ValidateUnitName(serviceName); err != nil {
		return fmt.Errorf("invalid service name %q: %w", serviceName, err)
	}

	// Get lifecycle from app
	lifecycle, err := app.GetLifecycle(ctx)
	if err != nil {
		return fmt.Errorf("failed to get lifecycle: %w", err)
	}

	// Get status from Lifecycle
	status, err := lifecycle.Status(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get status for service %q: %w", serviceName, err)
	}

	// Display status
	deps.Logger.Info("Service status",
		"service", serviceName,
		"active", status.Active,
		"state", status.State,
		"description", status.Description,
	)

	if status.SubState != "" {
		deps.Logger.Info("Substate", "substate", status.SubState)
	}

	if status.Active && status.PID > 0 {
		deps.Logger.Info("Service is running",
			"pid", status.PID,
			"since", status.Since,
		)
	}

	if status.Error != "" {
		deps.Logger.Error("Service error", "error", status.Error)
	}

	return nil
}

// buildDeps creates production dependencies for the status command.
func (c *StatusCommand) buildDeps(app *App) StatusDeps {
	return StatusDeps{
		CommonDeps: NewRootDeps(app),
	}
}
