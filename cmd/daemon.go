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
	"runtime"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

// DaemonOptions holds daemon command options.
type DaemonOptions struct {
	SyncInterval time.Duration
	RepoName     string
	Force        bool
}

// SyncRunner defines the interface for performing sync operations.
type SyncRunner interface {
	Run(context.Context, *App, SyncOptions, SyncDeps) error
	buildDeps(*App) SyncDeps
}

// DaemonDeps holds daemon dependencies.
type DaemonDeps struct {
	CommonDeps
	Notify      NotifyFunc
	SyncCommand SyncRunner
}

// DaemonCommand represents the daemon command for quad-ops CLI.
type DaemonCommand struct{}

// NewDaemonCommand creates a new DaemonCommand.
func NewDaemonCommand() *DaemonCommand {
	return &DaemonCommand{}
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

On Linux, the daemon integrates with systemd, sending readiness and watchdog 
notifications when running under systemd supervision. On macOS, the daemon runs 
without systemd integration.`,
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
	// Use platform-specific notify function
	var notifyFunc NotifyFunc
	if runtime.GOOS == "linux" {
		// Use systemd notifications on Linux
		notifyFunc = sdNotify
	} else {
		// No-op notifier on other platforms
		notifyFunc = func(_ bool, _ string) (bool, error) {
			return false, nil
		}
	}

	return DaemonDeps{
		CommonDeps:  NewRootDeps(app),
		Notify:      notifyFunc,
		SyncCommand: NewSyncCommand(),
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

	// Build sync dependencies once
	syncDeps := deps.SyncCommand.buildDeps(app)

	// Prepare sync options from daemon flags
	syncOpts := SyncOptions{
		RepoName: opts.RepoName,
		Force:    opts.Force,
		DryRun:   false,
	}

	// Perform initial sync
	if app.Config.Verbose {
		deps.Logger.Info("Performing initial sync")
	}

	if err := deps.SyncCommand.Run(ctx, app, syncOpts, syncDeps); err != nil {
		deps.Logger.Error("Initial sync failed", "error", err)
		// Continue to daemon mode even if initial sync fails
	}

	// Start daemon mode
	return c.runDaemon(ctx, app, syncOpts, deps, syncDeps)
}

// runDaemon starts the daemon loop with periodic sync operations.
func (c *DaemonCommand) runDaemon(ctx context.Context, app *App, syncOpts SyncOptions, deps DaemonDeps, syncDeps SyncDeps) error {
	deps.Logger.Info("Starting sync daemon", "interval", app.Config.SyncInterval)

	// Notify systemd that the daemon is ready (no-op on non-Linux)
	if sent, err := deps.Notify(false, SdNotifyReady); err != nil {
		deps.Logger.Warn("Failed to notify systemd of readiness", "error", err)
	} else if sent {
		deps.Logger.Info("Notified systemd that daemon is ready")
	}

	// Atomic guard to prevent overlapping syncs
	var syncing atomic.Bool

	// Backoff state for repeated failures
	consecutiveFailures := 0
	maxBackoffInterval := 30 * time.Minute
	baseInterval := app.Config.SyncInterval

	ticker := deps.Clock.Ticker(baseInterval)
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
			// Use atomic guard to prevent overlapping syncs
			if !syncing.CompareAndSwap(false, true) {
				deps.Logger.Warn("Previous sync still running, skipping this interval")
				continue
			}

			// Perform sync in current goroutine (blocking)
			deps.Logger.Debug("Starting scheduled sync")
			err := deps.SyncCommand.Run(ctx, app, syncOpts, syncDeps)

			// Release sync lock
			syncing.Store(false)

			if err != nil {
				consecutiveFailures++
				deps.Logger.Error("Sync failed", "error", err, "consecutive_failures", consecutiveFailures)

				// Apply exponential backoff on repeated failures
				if consecutiveFailures > 1 {
					// Calculate 2^(n-1), with bounds checking to prevent overflow
					exponent := consecutiveFailures - 1
					if exponent > 30 { // Prevent overflow beyond 2^30
						exponent = 30
					}
					backoffMultiplier := 1 << exponent // 2^(n-1)
					newInterval := baseInterval * time.Duration(backoffMultiplier)
					if newInterval > maxBackoffInterval {
						newInterval = maxBackoffInterval
					}

					deps.Logger.Warn("Applying backoff after repeated failures",
						"consecutive_failures", consecutiveFailures,
						"new_interval", newInterval)

					// Reset ticker with new interval
					ticker.Reset(newInterval)
				}
			} else {
				// Sync succeeded - reset failure counter and interval
				if consecutiveFailures > 0 {
					deps.Logger.Info("Sync succeeded after failures, resetting interval",
						"previous_failures", consecutiveFailures)
					consecutiveFailures = 0
					ticker.Reset(baseInterval)
				}
			}

		case <-watchdogTicker.C:
			// Send watchdog notification to systemd (no-op on non-Linux)
			if sent, err := deps.Notify(false, SdNotifyWatchdog); err != nil {
				deps.Logger.Debug("Failed to send watchdog notification", "error", err)
			} else if sent {
				deps.Logger.Debug("Sent watchdog notification to systemd")
			}
		}
	}
}
