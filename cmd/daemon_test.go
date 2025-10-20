package cmd

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestDaemonCommand_ValidationFailure tests system requirements validation.
func TestDaemonCommand_ValidationFailure(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return errors.New("system requirements not met")
			},
		}).
		Build(t)

	cmd := NewDaemonCommand().GetCobraCommand()
	SetupCommandContext(cmd, app)

	// PreRunE returns error when system requirements fail
	err := cmd.PreRunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system requirements not met")
}

// TestDaemonCommand_DirectoryCreationFailure tests quadlet directory creation failure.
func TestDaemonCommand_DirectoryCreationFailure(t *testing.T) {
	// Create mock sync command that never gets called
	mockSyncCmd := &SyncCommand{}

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error {
					return errors.New("permission denied")
				},
			},
			Logger: testutil.NewTestLogger(t),
		},
		SyncCommand: mockSyncCmd,
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
	var syncCount atomic.Int32

	// Create mock sync command that tracks calls
	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			syncCount.Add(1)
			return nil
		},
	}

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify:      func(_ bool, _ string) (bool, error) { return true, nil },
		SyncCommand: mockSyncCmd,
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			SyncInterval: 1 * time.Minute,
			QuadletDir:   "/tmp/test-quadlets",
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		SyncInterval: 1 * time.Minute,
	}

	// Use a context that times out immediately after initial sync
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should perform initial sync and then timeout
	err := daemonCmd.Run(ctx, app, opts, deps)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), syncCount.Load(), "Initial sync should have been performed")
}

// TestDaemonCommand_SystemdNotifications tests systemd notification behavior.
func TestDaemonCommand_SystemdNotifications(t *testing.T) {
	var notifyStates []string

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			return nil
		},
	}

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, state string) (bool, error) {
			notifyStates = append(notifyStates, state)
			return true, nil
		},
		SyncCommand: mockSyncCmd,
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
	assert.Contains(t, notifyStates, SdNotifyReady)
}

// TestDaemonCommand_SystemdNotificationError tests handling of systemd notification errors.
func TestDaemonCommand_SystemdNotificationError(t *testing.T) {
	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			return nil
		},
	}

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify: func(_ bool, _ string) (bool, error) {
			return false, errors.New("systemd not available")
		},
		SyncCommand: mockSyncCmd,
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
	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			return nil
		},
	}

	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify:      func(_ bool, _ string) (bool, error) { return true, nil },
		SyncCommand: mockSyncCmd,
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

// TestDaemonCommand_SyncFailureBackoff tests backoff on repeated sync failures.
func TestDaemonCommand_SyncFailureBackoff(t *testing.T) {
	var syncCalls atomic.Int32

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			syncCalls.Add(1)
			return errors.New("sync failed")
		},
	}

	mockClock := clock.NewMock()
	deps := DaemonDeps{
		CommonDeps: CommonDeps{
			Clock: mockClock,
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		Notify:      func(_ bool, _ string) (bool, error) { return true, nil },
		SyncCommand: mockSyncCmd,
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			SyncInterval: 1 * time.Minute,
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run daemon in background
	go func() {
		_ = daemonCmd.Run(ctx, app, opts, deps)
	}()

	// Wait a bit for initial sync
	time.Sleep(50 * time.Millisecond)

	// Initial sync should have failed
	assert.GreaterOrEqual(t, syncCalls.Load(), int32(1))

	// Note: Full backoff testing would require more complex timing control
	// This test verifies the basic mechanism is in place
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

// MockSyncCommand mocks SyncCommand for testing.
type MockSyncCommand struct {
	RunFunc       func(context.Context, *App, SyncOptions, SyncDeps) error
	buildDepsFunc func(*App) SyncDeps
}

func (m *MockSyncCommand) Run(ctx context.Context, app *App, opts SyncOptions, deps SyncDeps) error {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, app, opts, deps)
	}
	return nil
}

func (m *MockSyncCommand) buildDeps(app *App) SyncDeps {
	if m.buildDepsFunc != nil {
		return m.buildDepsFunc(app)
	}
	return SyncDeps{
		CommonDeps: NewRootDeps(app),
	}
}

func (m *MockSyncCommand) GetCobraCommand() *cobra.Command {
	return &cobra.Command{}
}
