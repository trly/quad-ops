package cmd

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

// MockSyncPerformer implements SyncPerformer for testing.
type MockSyncPerformer struct {
	PerformSyncFunc func(*App, *SyncCommand)
	CallCount       int
}

func (m *MockSyncPerformer) PerformSync(app *App, syncCmd *SyncCommand) {
	m.CallCount++
	if m.PerformSyncFunc != nil {
		m.PerformSyncFunc(app, syncCmd)
	}
}

// TestDaemonCommand_ValidationFailure tests system requirements failure.
func TestDaemonCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("systemd not found")
			},
		}).
		Build(t)

	cmd := NewDaemonCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	// PreRunE returns error instead of exiting
	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

// TestDaemonCommand_DirectoryCreationFailure tests quadlet directory creation failure.
func TestDaemonCommand_DirectoryCreationFailure(t *testing.T) {
	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error {
					return errors.New("permission denied")
				},
			},
			Logger: testutil.NewTestLogger(t),
		},
	}

	app := NewAppBuilder(t).Build(t)
	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{}

	err := daemonCmd.Run(context.Background(), app, opts, deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestDaemonCommand_InitialSync tests that initial sync is performed.
func TestDaemonCommand_InitialSync(t *testing.T) {
	var syncCount int

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, _ string) (bool, error) { return true, nil },
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			SyncInterval: 1 * time.Minute,
			QuadletDir:   "/tmp/test-quadlets",
		}).
		Build(t)

	// Override sync performer to count calls and cancel quickly
	daemonCmd := NewDaemonCommand()
	mockPerformer := &MockSyncPerformer{
		PerformSyncFunc: func(_ *App, _ *SyncCommand) {
			syncCount++
		},
	}
	daemonCmd.syncPerformer = mockPerformer

	opts := DaemonOptions{
		SyncInterval: 1 * time.Minute,
	}

	// Use a context that times out immediately after initial sync
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should perform initial sync and then timeout
	err := daemonCmd.Run(ctx, app, opts, deps)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, syncCount, "Initial sync should have been performed")
}

// TestDaemonCommand_SystemdNotifications tests systemd notification behavior.
func TestDaemonCommand_SystemdNotifications(t *testing.T) {
	var notifyStates []string

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, state string) (bool, error) {
			notifyStates = append(notifyStates, state)
			return true, nil
		},
	}

	app := NewAppBuilder(t).Build(t)
	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{}

	// Use a very short timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := daemonCmd.Run(ctx, app, opts, deps)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify systemd ready notification was sent
	assert.Contains(t, notifyStates, daemon.SdNotifyReady)
}

// TestDaemonCommand_SystemdNotificationError tests handling of systemd notification errors.
func TestDaemonCommand_SystemdNotificationError(t *testing.T) {
	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, _ string) (bool, error) {
			return false, errors.New("systemd not available")
		},
	}

	app := NewAppBuilder(t).Build(t)
	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{}

	// Test that daemon handles notification errors gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := daemonCmd.Run(ctx, app, opts, deps)
	// Should timeout, not fail due to systemd error
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestDaemonCommand_SyncIntervalOverride tests sync interval override.
func TestDaemonCommand_SyncIntervalOverride(t *testing.T) {
	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, _ string) (bool, error) { return true, nil },
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			SyncInterval: 5 * time.Minute, // Original interval
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		SyncInterval: 2 * time.Minute, // Override interval
	}

	// Use immediate timeout to just test the setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_ = daemonCmd.Run(ctx, app, opts, deps)

	// Verify sync interval was overridden in config
	assert.Equal(t, 2*time.Minute, app.Config.SyncInterval)
}

// TestDaemonCommand_Help tests help output.
func TestDaemonCommand_Help(t *testing.T) {
	cmd := NewDaemonCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Run quad-ops as a daemon")
	assert.Contains(t, output, "periodic synchronization")
	assert.Contains(t, output, "--sync-interval")
	assert.Contains(t, output, "--repo")
	assert.Contains(t, output, "--force")
}

// TestDaemonCommand_Flags tests command-specific flags.
func TestDaemonCommand_Flags(t *testing.T) {
	cmd := NewDaemonCommand().GetCobraCommand()

	// Test sync-interval flag
	syncIntervalFlag := cmd.Flags().Lookup("sync-interval")
	require.NotNil(t, syncIntervalFlag)
	assert.Equal(t, "5m0s", syncIntervalFlag.DefValue)

	// Test repo flag
	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag)
	assert.Equal(t, "", repoFlag.DefValue)

	// Test force flag
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

// TestDaemonCommand_SyncPerformer tests the sync performer interface.
func TestDaemonCommand_SyncPerformer(t *testing.T) {
	app := NewAppBuilder(t).Build(t)
	syncCmd := NewSyncCommand()
	daemonCmd := NewDaemonCommand()

	// Mock sync performer to avoid real operations
	mockPerformer := &MockSyncPerformer{
		PerformSyncFunc: func(receivedApp *App, receivedSyncCmd *SyncCommand) {
			// Verify the correct parameters were passed
			assert.Equal(t, app, receivedApp)
			assert.Equal(t, syncCmd, receivedSyncCmd)
		},
	}
	daemonCmd.syncPerformer = mockPerformer

	// Execute via the sync performer
	daemonCmd.syncPerformer.PerformSync(app, syncCmd)

	// Verify mock was called
	assert.Equal(t, 1, mockPerformer.CallCount)
}
