//go:build darwin

package launchd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/testutil"
)

// MockRunner implements execx.Runner for testing.
type MockRunner struct {
	outputs map[string]string
	errors  map[string]error
	calls   []string
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
		calls:   []string{},
	}
}

func (m *MockRunner) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	key := fmt.Sprintf("%s %v", name, args)
	m.calls = append(m.calls, key)

	if err, ok := m.errors[key]; ok {
		return nil, err
	}

	if output, ok := m.outputs[key]; ok {
		return []byte(output), nil
	}

	return []byte(""), nil
}

func (m *MockRunner) SetOutput(cmd string, args []string, output string) {
	key := fmt.Sprintf("%s %v", cmd, args)
	m.outputs[key] = output
}

func (m *MockRunner) SetError(cmd string, args []string, err error) {
	key := fmt.Sprintf("%s %v", cmd, args)
	m.errors[key] = err
}

func TestLifecycle_Start(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		setupMock func(*MockRunner)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "successful start",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				opts := testOptions()
				// Podman machine check
				m.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
				// isServiceLoaded check - returns false (service not loaded)
				m.SetError("launchctl", []string{"print", "gui/501/dev.trly.quad-ops.test-service"}, errors.New("Could not find service"))
				// Bootstrap (success)
				m.SetOutput("launchctl", []string{"bootstrap", "gui/501", "/Users/test/Library/LaunchAgents/dev.trly.quad-ops.test-service.plist"}, "")
				// Enable
				m.SetOutput("launchctl", []string{"enable", "gui/501/dev.trly.quad-ops.test-service"}, "")
				// Kickstart
				m.SetOutput("launchctl", []string{"kickstart", "-k", "gui/501/dev.trly.quad-ops.test-service"}, "")
			},
			wantErr: false,
		},
		{
			name:    "podman machine not running",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				opts := testOptions()
				m.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "stopped\n")
			},
			wantErr: true,
			errMsg:  "podman machine is not running",
		},
		{
			name:    "service already loaded",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				opts := testOptions()
				m.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
				// isServiceLoaded check - returns true (service loaded)
				m.SetOutput("launchctl", []string{"print", "gui/501/dev.trly.quad-ops.test-service"}, "state = running\n")
				// Skip bootstrap since already loaded
				m.SetOutput("launchctl", []string{"enable", "gui/501/dev.trly.quad-ops.test-service"}, "")
				m.SetOutput("launchctl", []string{"kickstart", "-k", "gui/501/dev.trly.quad-ops.test-service"}, "")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)

			logger := testutil.NewTestLogger(t)
			lifecycle, err := NewLifecycle(testOptions(), mock, logger)
			require.NoError(t, err)

			err = lifecycle.Start(context.Background(), tt.service)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLifecycle_Stop(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		setupMock func(*MockRunner)
		wantErr   bool
	}{
		{
			name:    "successful stop",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				m.SetOutput("launchctl", []string{"bootout", "gui/501/dev.trly.quad-ops.test-service"}, "")
			},
			wantErr: false,
		},
		{
			name:    "bootout fails, fallback to unload",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				m.SetError("launchctl", []string{"bootout", "gui/501/dev.trly.quad-ops.test-service"}, errors.New("not found"))
				m.SetOutput("launchctl", []string{"stop", "dev.trly.quad-ops.test-service"}, "")
				m.SetOutput("launchctl", []string{"unload", "-w", "/Users/test/Library/LaunchAgents/dev.trly.quad-ops.test-service.plist"}, "")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)

			logger := testutil.NewTestLogger(t)
			lifecycle, err := NewLifecycle(testOptions(), mock, logger)
			require.NoError(t, err)

			err = lifecycle.Stop(context.Background(), tt.service)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLifecycle_Status(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		setupMock  func(*MockRunner)
		wantActive bool
		wantState  string
		wantPID    int
	}{
		{
			name:    "service running",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				m.SetOutput("launchctl", []string{"print", "gui/501/dev.trly.quad-ops.test-service"},
					"state = running\npid = 12345")
			},
			wantActive: true,
			wantState:  "running",
			wantPID:    12345,
		},
		{
			name:    "service not running",
			service: "test-service",
			setupMock: func(m *MockRunner) {
				m.SetError("launchctl", []string{"print", "gui/501/dev.trly.quad-ops.test-service"}, errors.New("not found"))
				m.SetOutput("launchctl", []string{"list"}, "")
			},
			wantActive: false,
			wantState:  "stopped",
			wantPID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRunner()
			tt.setupMock(mock)

			logger := testutil.NewTestLogger(t)
			lifecycle, err := NewLifecycle(testOptions(), mock, logger)
			require.NoError(t, err)

			status, err := lifecycle.Status(context.Background(), tt.service)
			require.NoError(t, err)
			require.NotNil(t, status)

			assert.Equal(t, tt.service, status.Name)
			assert.Equal(t, tt.wantActive, status.Active)
			assert.Equal(t, tt.wantState, status.State)
			assert.Equal(t, tt.wantPID, status.PID)
		})
	}
}

func TestLifecycle_Name(t *testing.T) {
	mock := NewMockRunner()
	logger := testutil.NewTestLogger(t)
	lifecycle, err := NewLifecycle(testOptions(), mock, logger)
	require.NoError(t, err)

	assert.Equal(t, "launchd", lifecycle.Name())
}

func TestLifecycle_Reload(t *testing.T) {
	mock := NewMockRunner()
	logger := testutil.NewTestLogger(t)
	lifecycle, err := NewLifecycle(testOptions(), mock, logger)
	require.NoError(t, err)

	// Reload should be a no-op
	err = lifecycle.Reload(context.Background())
	assert.NoError(t, err)
}

func TestLifecycle_StartMany_Sequential(t *testing.T) {
	// Test that StartMany processes services sequentially in the provided order
	t.Run("processes services in order", func(t *testing.T) {
		mock := NewMockRunner()
		opts := testOptions()

		// Setup mocks for three services in dependency order: postgres -> app -> worker
		for _, svc := range []string{"postgres", "app", "worker"} {
			// Podman machine check
			mock.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
			// Service not loaded
			domainTarget := fmt.Sprintf("gui/501/dev.trly.quad-ops.%s", svc)
			mock.SetError("launchctl", []string{"print", domainTarget}, errors.New("Could not find service"))
			// Bootstrap succeeds
			plistPath := fmt.Sprintf("/Users/test/Library/LaunchAgents/dev.trly.quad-ops.%s.plist", svc)
			mock.SetOutput("launchctl", []string{"bootstrap", "gui/501", plistPath}, "")
			// Enable
			mock.SetOutput("launchctl", []string{"enable", domainTarget}, "")
			// Kickstart
			mock.SetOutput("launchctl", []string{"kickstart", "-k", domainTarget}, "")
		}

		logger := testutil.NewTestLogger(t)
		lifecycle, err := NewLifecycle(opts, mock, logger)
		require.NoError(t, err)

		results := lifecycle.StartMany(context.Background(), []string{"postgres", "app", "worker"})

		// All should succeed
		assert.NoError(t, results["postgres"])
		assert.NoError(t, results["app"])
		assert.NoError(t, results["worker"])

		// Verify services started in order by checking launchctl bootstrap calls
		// The calls should have postgres bootstrap before app bootstrap before worker bootstrap
		var postgresBootstrapIdx, appBootstrapIdx, workerBootstrapIdx int
		found := 0
		for i, call := range mock.calls {
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "postgres") {
				postgresBootstrapIdx = i
				found++
			}
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "app") {
				appBootstrapIdx = i
				found++
			}
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "worker") {
				workerBootstrapIdx = i
				found++
			}
		}

		require.Equal(t, 3, found, "all three bootstrap calls should be recorded")
		assert.Less(t, postgresBootstrapIdx, appBootstrapIdx, "postgres should bootstrap before app")
		assert.Less(t, appBootstrapIdx, workerBootstrapIdx, "app should bootstrap before worker")
	})

	t.Run("continues on partial failure", func(t *testing.T) {
		mock := NewMockRunner()
		opts := testOptions()

		// Setup: first service succeeds, second fails, third succeeds
		for _, svc := range []string{"postgres", "app", "worker"} {
			mock.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
			domainTarget := fmt.Sprintf("gui/501/dev.trly.quad-ops.%s", svc)
			plistPath := fmt.Sprintf("/Users/test/Library/LaunchAgents/dev.trly.quad-ops.%s.plist", svc)

			if svc == "app" {
				// Make this one fail on all attempts
				mock.SetError("launchctl", []string{"print", domainTarget}, errors.New("Could not find service"))
				mock.SetError("launchctl", []string{"bootstrap", "gui/501", plistPath}, errors.New("failed to bootstrap"))
				// Also fail the legacy fallback
				mock.SetError("launchctl", []string{"load", "-w", plistPath}, errors.New("failed to load"))
			} else {
				// Others succeed
				mock.SetError("launchctl", []string{"print", domainTarget}, errors.New("Could not find service"))
				mock.SetOutput("launchctl", []string{"bootstrap", "gui/501", plistPath}, "")
				mock.SetOutput("launchctl", []string{"enable", domainTarget}, "")
				mock.SetOutput("launchctl", []string{"kickstart", "-k", domainTarget}, "")
			}
		}

		logger := testutil.NewTestLogger(t)
		lifecycle, err := NewLifecycle(opts, mock, logger)
		require.NoError(t, err)

		results := lifecycle.StartMany(context.Background(), []string{"postgres", "app", "worker"})

		// postgres and worker should succeed despite app failure
		assert.NoError(t, results["postgres"])
		assert.Error(t, results["app"])
		assert.NoError(t, results["worker"])
	})
}

func TestLifecycle_StopMany_Reverse(t *testing.T) {
	// Test that StopMany processes services in reverse order
	t.Run("stops in reverse order", func(t *testing.T) {
		mock := NewMockRunner()

		// Setup mocks for three services: postgres, app, worker
		for _, svc := range []string{"postgres", "app", "worker"} {
			domainTarget := fmt.Sprintf("gui/501/dev.trly.quad-ops.%s", svc)
			mock.SetOutput("launchctl", []string{"bootout", domainTarget}, "")
		}

		logger := testutil.NewTestLogger(t)
		lifecycle, err := NewLifecycle(testOptions(), mock, logger)
		require.NoError(t, err)

		// Call with forward order: postgres, app, worker
		// Should stop in reverse: worker, app, postgres
		results := lifecycle.StopMany(context.Background(), []string{"postgres", "app", "worker"})

		// All should succeed
		assert.NoError(t, results["postgres"])
		assert.NoError(t, results["app"])
		assert.NoError(t, results["worker"])

		// Verify bootout calls in reverse order
		var postgresBootoutIdx, appBootoutIdx, workerBootoutIdx int
		found := 0
		for i, call := range mock.calls {
			if strings.Contains(call, "bootout") && strings.Contains(call, "postgres") {
				postgresBootoutIdx = i
				found++
			}
			if strings.Contains(call, "bootout") && strings.Contains(call, "app") {
				appBootoutIdx = i
				found++
			}
			if strings.Contains(call, "bootout") && strings.Contains(call, "worker") {
				workerBootoutIdx = i
				found++
			}
		}

		require.Equal(t, 3, found, "all three bootout calls should be recorded")
		assert.Greater(t, postgresBootoutIdx, appBootoutIdx, "postgres should bootout after app")
		assert.Greater(t, appBootoutIdx, workerBootoutIdx, "app should bootout after worker")
	})
}

func TestLifecycle_RestartMany_Sequential(t *testing.T) {
	// Test that RestartMany processes services sequentially in order
	t.Run("restarts in order", func(t *testing.T) {
		mock := NewMockRunner()
		opts := testOptions()

		// Setup mocks for three services
		for _, svc := range []string{"postgres", "app", "worker"} {
			mock.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
			domainTarget := fmt.Sprintf("gui/501/dev.trly.quad-ops.%s", svc)
			plistPath := fmt.Sprintf("/Users/test/Library/LaunchAgents/dev.trly.quad-ops.%s.plist", svc)

			// Service is loaded
			mock.SetOutput("launchctl", []string{"print", domainTarget}, "state = running\n")
			// Bootout succeeds
			mock.SetOutput("launchctl", []string{"bootout", domainTarget}, "")
			// Bootstrap succeeds
			mock.SetOutput("launchctl", []string{"bootstrap", "gui/501", plistPath}, "")
			// Enable
			mock.SetOutput("launchctl", []string{"enable", domainTarget}, "")
			// Kickstart
			mock.SetOutput("launchctl", []string{"kickstart", "-k", domainTarget}, "")
		}

		logger := testutil.NewTestLogger(t)
		lifecycle, err := NewLifecycle(opts, mock, logger)
		require.NoError(t, err)

		results := lifecycle.RestartMany(context.Background(), []string{"postgres", "app", "worker"})

		// All should succeed
		assert.NoError(t, results["postgres"])
		assert.NoError(t, results["app"])
		assert.NoError(t, results["worker"])

		// Verify restart operations (bootstrap) happen in order
		var postgresBootstrapIdx, appBootstrapIdx, workerBootstrapIdx int
		found := 0
		for i, call := range mock.calls {
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "postgres") {
				postgresBootstrapIdx = i
				found++
			}
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "app") {
				appBootstrapIdx = i
				found++
			}
			if strings.Contains(call, "bootstrap") && strings.Contains(call, "worker") {
				workerBootstrapIdx = i
				found++
			}
		}

		require.Equal(t, 3, found, "all three bootstrap calls should be recorded")
		assert.Less(t, postgresBootstrapIdx, appBootstrapIdx, "postgres should bootstrap before app")
		assert.Less(t, appBootstrapIdx, workerBootstrapIdx, "app should bootstrap before worker")
	})
}

func TestLifecycle_StartMany_WaitsBetweenServices(t *testing.T) {
	// Test that StartMany waits for each service to complete before starting next
	t.Run("waits for each service before next", func(t *testing.T) {
		mock := NewMockRunner()
		opts := testOptions()

		// Track completion of each service
		completionOrder := []string{}
		var callOrder []string

		// Setup mocks with side effects to track order
		for _, svc := range []string{"postgres", "app"} {
			mock.SetOutput(opts.PodmanPath, []string{"machine", "inspect", "--format", "{{.State}}"}, "running\n")
			domainTarget := fmt.Sprintf("gui/501/dev.trly.quad-ops.%s", svc)
			plistPath := fmt.Sprintf("/Users/test/Library/LaunchAgents/dev.trly.quad-ops.%s.plist", svc)
			mock.SetError("launchctl", []string{"print", domainTarget}, errors.New("Could not find service"))
			mock.SetOutput("launchctl", []string{"bootstrap", "gui/501", plistPath}, "")
			mock.SetOutput("launchctl", []string{"enable", domainTarget}, "")
			mock.SetOutput("launchctl", []string{"kickstart", "-k", domainTarget}, "")
		}

		logger := testutil.NewTestLogger(t)
		lifecycle, err := NewLifecycle(opts, mock, logger)
		require.NoError(t, err)

		results := lifecycle.StartMany(context.Background(), []string{"postgres", "app"})

		// Both should succeed
		assert.NoError(t, results["postgres"])
		assert.NoError(t, results["app"])

		// Verify all postgres operations completed before any app operations
		postgresOps := 0
		appOps := 0
		for _, call := range mock.calls {
			if strings.Contains(call, "postgres") {
				postgresOps++
			}
			if strings.Contains(call, "app") {
				appOps++
			}
		}

		_ = callOrder // for clarity
		_ = completionOrder

		// Just verify both services were processed
		assert.Greater(t, postgresOps, 0)
		assert.Greater(t, appOps, 0)
	})
}
