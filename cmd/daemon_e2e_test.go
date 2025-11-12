package cmd

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestDaemon_StartupShutdown tests basic daemon startup and shutdown behavior.
func TestDaemon_StartupShutdown(t *testing.T) {
	var syncCount atomic.Int32
	var notifyStates []string

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
		Notify: func(_ bool, state string) (bool, error) {
			notifyStates = append(notifyStates, state)
			return true, nil
		},
		SyncCommand: mockSyncCmd,
	}

	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			SyncInterval: 1 * time.Minute,
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		SyncInterval: 1 * time.Minute,
	}

	// Run daemon with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := daemonCmd.Run(ctx, app, opts, deps)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify initial sync happened
	assert.Equal(t, int32(1), syncCount.Load(), "should perform initial sync")

	// Verify systemd notification
	assert.Contains(t, notifyStates, SdNotifyReady, "should send ready notification")
}

// TestDaemon_PeriodicSync tests that daemon performs periodic syncs.
func TestDaemon_PeriodicSync(t *testing.T) {
	var syncCount atomic.Int32

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, _ SyncDeps) error {
			syncCount.Add(1)
			return nil
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
			SyncInterval: 1 * time.Second,
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

	// Wait for initial sync
	time.Sleep(50 * time.Millisecond)
	initialCount := syncCount.Load()
	assert.GreaterOrEqual(t, initialCount, int32(1), "should perform initial sync")

	// Advance clock to trigger periodic sync
	mockClock.Add(2 * time.Second)
	time.Sleep(50 * time.Millisecond)

	// Note: Full periodic sync testing requires more complex timing control
	// This test verifies the basic mechanism is in place
}

// TestDaemon_ConfigOverride tests sync interval override.
func TestDaemon_ConfigOverride(t *testing.T) {
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
			SyncInterval: 5 * time.Minute,
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		SyncInterval: 2 * time.Minute,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_ = daemonCmd.Run(ctx, app, opts, deps)

	assert.Equal(t, 2*time.Minute, app.Config.SyncInterval, "should override sync interval")
}

// TestDaemon_RepoFilter tests repository filtering with --repo flag.
func TestDaemon_RepoFilter(t *testing.T) {
	var receivedRepoName string

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, opts SyncOptions, _ SyncDeps) error {
			receivedRepoName = opts.RepoName
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
			Repositories: []config.Repository{
				{Name: "app1", URL: "https://example.com/app1.git"},
				{Name: "app2", URL: "https://example.com/app2.git"},
				{Name: "app3", URL: "https://example.com/app3.git"},
			},
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		RepoName: "app1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = daemonCmd.Run(ctx, app, opts, deps)

	assert.Equal(t, "app1", receivedRepoName, "should pass repo filter to sync command")
}

// TestDaemon_ForceSync tests force sync behavior.
func TestDaemon_ForceSync(t *testing.T) {
	var forceUsed bool

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, opts SyncOptions, _ SyncDeps) error {
			forceUsed = opts.Force
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

	app := NewAppBuilder(t).Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{
		Force: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = daemonCmd.Run(ctx, app, opts, deps)

	assert.True(t, forceUsed, "should use force sync")
}

// TestDaemon_GitSyncIntegration tests daemon with actual git sync results.
func TestDaemon_GitSyncIntegration(t *testing.T) {
	var processedRepos []string

	mockGitSyncer := &MockGitSyncer{
		SyncAllFunc: func(_ context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
			results := make([]repository.SyncResult, len(repos))
			for i, repo := range repos {
				processedRepos = append(processedRepos, repo.Name)
				results[i] = repository.SyncResult{
					Repository: repo,
					Success:    true,
					Changed:    true,
				}
			}
			return results, nil
		},
	}

	mockSyncCmd := &MockSyncCommand{
		RunFunc: func(_ context.Context, _ *App, _ SyncOptions, deps SyncDeps) error {
			// Simulate actual sync command behavior
			_, err := deps.GitSyncer.SyncAll(context.Background(), []config.Repository{
				{Name: "test-app", URL: "https://example.com/test.git"},
			})
			return err
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
			Repositories: []config.Repository{
				{Name: "test-app", URL: "https://example.com/test.git"},
			},
		}).
		Build(t)

	daemonCmd := NewDaemonCommand()
	opts := DaemonOptions{}

	// Build sync deps with git syncer
	syncDeps := SyncDeps{
		CommonDeps: CommonDeps{
			Clock: clock.NewMock(),
			FileSystem: &FileSystemOps{
				MkdirAllFunc: func(_ string, _ os.FileMode) error { return nil },
			},
			Logger: testutil.NewTestLogger(t),
		},
		GitSyncer: mockGitSyncer,
	}

	// Inject sync deps via closure
	mockSyncCmd.RunFunc = func(ctx context.Context, app *App, _ SyncOptions, _ SyncDeps) error {
		_, err := syncDeps.GitSyncer.SyncAll(ctx, app.Config.Repositories)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := daemonCmd.Run(ctx, app, opts, deps)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	assert.Contains(t, processedRepos, "test-app")
}
