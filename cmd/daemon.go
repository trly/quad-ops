// Package cmd provides the command line interface for quad-ops
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
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"
)

// DaemonOptions holds daemon command options.
type DaemonOptions struct {
	SyncInterval time.Duration
	RepoName     string
	Force        bool
}

// DaemonDeps holds daemon dependencies.
type DaemonDeps struct {
	CommonDeps
	Notify NotifyFunc
}

// SyncPerformer defines the interface for performing sync operations.
type SyncPerformer interface {
	PerformSync(context.Context, *App, *SyncCommand, SyncOptions, SyncDeps, DaemonDeps)
}

// DefaultSyncPerformer implements SyncPerformer with the default behavior.
type DefaultSyncPerformer struct{}

// PerformSync executes a sync operation using the default implementation.
func (d *DefaultSyncPerformer) PerformSync(ctx context.Context, app *App, syncCmd *SyncCommand, opts SyncOptions, syncDeps SyncDeps, daemonDeps DaemonDeps) {
	if err := syncCmd.syncRepositories(ctx, app, opts, syncDeps); err != nil {
		daemonDeps.Logger.Error("Sync failed", "error", err)
	}
}

// DaemonCommand represents the daemon command for quad-ops CLI.
type DaemonCommand struct {
	// syncPerformer allows tests to override sync behavior
	syncPerformer SyncPerformer
}

// NewDaemonCommand creates a new DaemonCommand.
func NewDaemonCommand() *DaemonCommand {
	cmd := &DaemonCommand{}
	// Set default sync performer implementation
	cmd.syncPerformer = &DefaultSyncPerformer{}
	return cmd
}

// getApp retrieves the App from the command context.
func (c *DaemonCommand) getApp(cmd *cobra.Command) *App {
	return cmd.Context().Value(appContextKey).(*App)
}

// GetCobraCommand returns the cobra command for daemon operations.
func (c *DaemonCommand) GetCobraCommand() *cobra.Command {
	var opts DaemonOptions

	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run quad-ops as a daemon with periodic synchronization",
		Long: `Run quad-ops as a daemon with periodic synchronization of configured repositories.

The daemon will perform initial synchronization and then continue running, 
periodically syncing repositories at the specified interval. This is ideal 
for continuous deployment scenarios where you want automatic updates.

The daemon integrates with systemd, sending readiness and watchdog notifications
when running under systemd supervision.`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			return app.Validator.SystemRequirements()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := c.getApp(cmd)
			deps := c.buildDeps(app)
			return c.Run(cmd.Context(), app, opts, deps)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	daemonCmd.Flags().DurationVarP(&opts.SyncInterval, "sync-interval", "i", 5*time.Minute, "Interval between synchronization checks")
	daemonCmd.Flags().StringVarP(&opts.RepoName, "repo", "r", "", "Synchronize a single, named, repository")
	daemonCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force synchronization even if repository has not changed")

	return daemonCmd
}

// buildDeps creates production dependencies for the daemon.
func (c *DaemonCommand) buildDeps(app *App) DaemonDeps {
	return DaemonDeps{
		CommonDeps: NewRootDeps(app),
		Notify:     daemon.SdNotify,
	}
}

// Run executes the daemon with injected dependencies.
func (c *DaemonCommand) Run(ctx context.Context, app *App, opts DaemonOptions, deps DaemonDeps) error {
	// Ensure quadlet directory exists
	if err := deps.FileSystem.MkdirAll(app.Config.QuadletDir, 0750); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	// Override sync interval if specified
	if opts.SyncInterval > 0 {
		app.Config.SyncInterval = opts.SyncInterval
	}

	// Create sync command instance and build its dependencies once
	syncCmd := NewSyncCommand()
	syncDeps := syncCmd.buildDeps(app)

	// Prepare sync options from daemon flags
	syncOpts := SyncOptions{
		RepoName: opts.RepoName,
		Force:    opts.Force,
	}

	// Perform initial sync
	if app.Config.Verbose {
		deps.Logger.Info("Performing initial sync")
	}
	c.syncPerformer.PerformSync(ctx, app, syncCmd, syncOpts, syncDeps, deps)

	// Start daemon mode
	return c.runDaemon(ctx, app, syncCmd, syncOpts, syncDeps, deps)
}

// runDaemon starts the daemon loop with periodic sync operations.
func (c *DaemonCommand) runDaemon(ctx context.Context, app *App, syncCmd *SyncCommand, syncOpts SyncOptions, syncDeps SyncDeps, deps DaemonDeps) error {
	deps.Logger.Info("Starting sync daemon", "interval", app.Config.SyncInterval)

	// Notify systemd that the daemon is ready
	if sent, err := deps.Notify(false, daemon.SdNotifyReady); err != nil {
		deps.Logger.Warn("Failed to notify systemd of readiness", "error", err)
	} else if sent {
		deps.Logger.Info("Notified systemd that daemon is ready")
	}

	ticker := deps.Clock.Ticker(app.Config.SyncInterval)
	defer ticker.Stop()

	// Send periodic watchdog notifications if configured
	watchdogTicker := deps.Clock.Ticker(30 * time.Second)
	defer watchdogTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			deps.Logger.Info("Daemon context cancelled, shutting down")
			return ctx.Err()
		case <-ticker.C:
			deps.Logger.Debug("Starting scheduled sync")
			c.syncPerformer.PerformSync(ctx, app, syncCmd, syncOpts, syncDeps, deps)
		case <-watchdogTicker.C:
			// Send watchdog notification to systemd
			if sent, err := deps.Notify(false, daemon.SdNotifyWatchdog); err != nil {
				deps.Logger.Debug("Failed to send watchdog notification", "error", err)
			} else if sent {
				deps.Logger.Debug("Sent watchdog notification to systemd")
			}
		}
	}
}
