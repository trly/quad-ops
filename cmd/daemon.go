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
	"os"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"
)

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

var (
	daemonSyncInterval time.Duration
	daemonRepoName     string
	daemonForce        bool
)

// GetCobraCommand returns the cobra command for daemon operations.
func (c *DaemonCommand) GetCobraCommand() *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run quad-ops as a daemon with periodic synchronization",
		Long: `Run quad-ops as a daemon with periodic synchronization of configured repositories.

The daemon will perform initial synchronization and then continue running, 
periodically syncing repositories at the specified interval. This is ideal 
for continuous deployment scenarios where you want automatic updates.

The daemon integrates with systemd, sending readiness and watchdog notifications
when running under systemd supervision.`,
		PreRun: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)
			// Validate system requirements for daemon mode
			if err := app.Validator.SystemRequirements(); err != nil {
				app.Logger.Error("System requirements not met", "error", err)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			app := c.getApp(cmd)

			// Ensure quadlet directory exists
			if err := os.MkdirAll(app.Config.QuadletDir, 0750); err != nil {
				app.Logger.Error("Failed to create quadlet directory", "error", err)
				os.Exit(1)
			}

			// Override sync interval if specified
			if daemonSyncInterval > 0 {
				app.Config.SyncInterval = daemonSyncInterval
			}

			// Create sync command instance for reuse
			syncCmd := NewSyncCommand()

			// Perform initial sync
			if app.Config.Verbose {
				app.Logger.Info("Performing initial sync")
			}
			c.performSync(app, syncCmd)

			// Start daemon mode
			c.runDaemon(app, syncCmd)
		},
	}

	daemonCmd.Flags().DurationVarP(&daemonSyncInterval, "sync-interval", "i", 5*time.Minute, "Interval between synchronization checks")
	daemonCmd.Flags().StringVarP(&daemonRepoName, "repo", "r", "", "Synchronize a single, named, repository")
	daemonCmd.Flags().BoolVarP(&daemonForce, "force", "f", false, "Force synchronization even if repository has not changed")

	return daemonCmd
}

// performSync executes a single sync operation using the provided sync command.
func (c *DaemonCommand) performSync(app *App, syncCmd *SyncCommand) {
	// Temporarily set global variables for sync command compatibility
	oldRepoName := repoName
	oldForce := force

	repoName = daemonRepoName
	force = daemonForce

	// Execute sync
	syncCmd.syncRepositories(app)

	// Restore original values
	repoName = oldRepoName
	force = oldForce
}

// runDaemon starts the daemon loop with periodic sync operations.
func (c *DaemonCommand) runDaemon(app *App, syncCmd *SyncCommand) {
	app.Logger.Info("Starting sync daemon", "interval", app.Config.SyncInterval)

	// Notify systemd that the daemon is ready
	if sent, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		app.Logger.Warn("Failed to notify systemd of readiness", "error", err)
	} else if sent {
		app.Logger.Info("Notified systemd that daemon is ready")
	}

	ticker := time.NewTicker(app.Config.SyncInterval)
	defer ticker.Stop()

	// Send periodic watchdog notifications if configured
	watchdogTicker := time.NewTicker(30 * time.Second)
	defer watchdogTicker.Stop()

	for {
		select {
		case <-ticker.C:
			app.Logger.Debug("Starting scheduled sync")
			c.performSync(app, syncCmd)
		case <-watchdogTicker.C:
			// Send watchdog notification to systemd
			if sent, err := daemon.SdNotify(false, daemon.SdNotifyWatchdog); err != nil {
				app.Logger.Debug("Failed to send watchdog notification", "error", err)
			} else if sent {
				app.Logger.Debug("Sent watchdog notification to systemd")
			}
		}
	}
}
