//go:build darwin

package launchd

import (
	"context"
	"errors"
	"fmt"
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
