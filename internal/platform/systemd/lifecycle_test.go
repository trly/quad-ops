// Package systemd provides systemd-specific platform implementations.
package systemd

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	dbusapi "github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestLifecycle_Name(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	l := NewLifecycle(nil, nil, false, logger)

	assert.Equal(t, "systemd", l.Name())
}

func TestLifecycle_cleanupOrphanedRootlessportProcesses(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	l := NewLifecycle(nil, nil, false, logger)

	ctx := context.Background()

	// This test mainly ensures the function doesn't panic and handles errors gracefully
	// In a real environment, it would check for and clean up rootlessport processes
	err := l.cleanupOrphanedRootlessportProcesses(ctx)

	// The function should not return an error even if pgrep/kill fail
	// It should log warnings instead of failing the operation
	assert.NoError(t, err)
}

// MockConnectionFactory creates connections that can be configured to succeed/fail.
type MockConnectionFactory struct {
	attemptCount int
	failAttempts int // Fail first N attempts
	connection   systemd.Connection
}

func (m *MockConnectionFactory) NewConnection(_ context.Context, _ bool) (systemd.Connection, error) {
	m.attemptCount++
	return m.connection, nil
}

// MockConnection simulates a systemd D-Bus connection.
type MockConnection struct {
	factory  *MockConnectionFactory // Reference to track attempt count
	props    map[string]interface{}
	failMode int // 0=never, 1=always, 2=for first N attempts then succeed
}

func (m *MockConnection) GetUnitProperties(_ context.Context, _ string) (map[string]interface{}, error) {
	switch m.failMode {
	case 0: // Never fail
		return m.props, nil
	case 1: // Always fail
		return nil, errors.New("unit not found")
	case 2: // Fail first N attempts
		if m.factory.attemptCount <= m.factory.failAttempts {
			return nil, errors.New("unit not found")
		}
		return m.props, nil
	}
	return m.props, nil
}

func (m *MockConnection) GetUnitProperty(_ context.Context, _, _ string) (*dbusapi.Property, error) {
	return nil, nil
}

func (m *MockConnection) ResetFailedUnit(_ context.Context, _ string) error {
	return nil
}

func (m *MockConnection) StartUnit(_ context.Context, _, _ string) (chan string, error) {
	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockConnection) StopUnit(_ context.Context, _, _ string) (chan string, error) {
	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockConnection) RestartUnit(_ context.Context, _, _ string) (chan string, error) {
	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockConnection) Reload(_ context.Context) error {
	return nil
}

func (m *MockConnection) Close() error {
	return nil
}

// MockRestartOrderConnection tracks the order of restart calls.
type MockRestartOrderConnection struct {
	baseProps    map[string]interface{}
	restartOrder *[]string
	mu           *sync.Mutex
}

func (m *MockRestartOrderConnection) GetUnitProperties(_ context.Context, _ string) (map[string]interface{}, error) {
	return m.baseProps, nil
}

func (m *MockRestartOrderConnection) GetUnitProperty(_ context.Context, _, _ string) (*dbusapi.Property, error) {
	return nil, nil
}

func (m *MockRestartOrderConnection) ResetFailedUnit(_ context.Context, _ string) error {
	return nil
}

func (m *MockRestartOrderConnection) StartUnit(_ context.Context, _, _ string) (chan string, error) {
	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockRestartOrderConnection) StopUnit(_ context.Context, _, _ string) (chan string, error) {
	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockRestartOrderConnection) RestartUnit(_ context.Context, name, _ string) (chan string, error) {
	m.mu.Lock()
	*m.restartOrder = append(*m.restartOrder, name)
	m.mu.Unlock()

	ch := make(chan string, 1)
	ch <- "done"
	close(ch)
	return ch, nil
}

func (m *MockRestartOrderConnection) Reload(_ context.Context) error {
	return nil
}

func (m *MockRestartOrderConnection) Close() error {
	return nil
}

func TestLifecycle_waitForUnitGeneration_Success(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{
		factory:  mockFactory,
		props:    map[string]interface{}{"LoadState": "loaded"},
		failMode: 0,
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	err := l.waitForUnitGeneration(ctx, "test.service")

	assert.NoError(t, err)
	assert.Equal(t, 1, mockFactory.attemptCount)
}

func TestLifecycle_waitForUnitGeneration_RetrySucceeds(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockFactory := &MockConnectionFactory{failAttempts: 2}
	// Fail first 2 attempts, succeed on 3rd
	mockConn := &MockConnection{
		factory:  mockFactory,
		props:    map[string]interface{}{"LoadState": "loaded"},
		failMode: 2,
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	err := l.waitForUnitGeneration(ctx, "test.service")

	assert.NoError(t, err)
	assert.Equal(t, 3, mockFactory.attemptCount)
}

func TestLifecycle_waitForUnitGeneration_Timeout(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{factory: mockFactory, failMode: 1} // Always fail
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(100 * time.Millisecond) // Very short timeout for testing

	ctx := context.Background()
	err := l.waitForUnitGeneration(ctx, "test.service")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reach 'loaded' state")
	assert.Contains(t, err.Error(), "100ms")
}

func TestLifecycle_waitForUnitGeneration_ContextCancellation(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{factory: mockFactory, failMode: 1} // Always fail
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(10 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := l.waitForUnitGeneration(ctx, "test.service")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestLifecycle_SetUnitGenerationTimeout(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	l := NewLifecycle(nil, nil, false, logger)

	// Default should be 5 seconds
	assert.Equal(t, 5*time.Second, l.unitGenerationTimeout)

	// Set a custom timeout
	l.SetUnitGenerationTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, l.unitGenerationTimeout)
}

func TestLifecycle_RestartMany_UnitAvailableForRestart(t *testing.T) {
	// This test verifies that after waitForUnitGeneration succeeds,
	// RestartUnit operations also succeed immediately (not flaky)
	logger := testutil.NewTestLogger(t)

	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{
		factory:  mockFactory,
		props:    map[string]interface{}{"LoadState": "loaded"},
		failMode: 0, // GetUnitProperties succeeds immediately
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	results := l.RestartMany(ctx, []string{"test-svc"})

	// Should succeed without "Unit not found" errors
	assert.Len(t, results, 1)
	assert.NoError(t, results["test-svc"])
}

func TestLifecycle_RestartMany_SequentialExecution(t *testing.T) {
	// Verifies RestartMany processes services sequentially in provided order
	logger := testutil.NewTestLogger(t)

	// Track restart order
	var restartOrder []string
	var mu sync.Mutex

	mockFactory := &MockConnectionFactory{}
	mockConn := &MockRestartOrderConnection{
		baseProps:    map[string]interface{}{"LoadState": "loaded"},
		restartOrder: &restartOrder,
		mu:           &mu,
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	services := []string{"svc-a", "svc-b", "svc-c"}
	results := l.RestartMany(ctx, services)

	// All should succeed
	assert.Len(t, results, 3)
	assert.NoError(t, results["svc-a"])
	assert.NoError(t, results["svc-b"])
	assert.NoError(t, results["svc-c"])

	// Verify sequential order (exact match)
	assert.Equal(t, []string{"svc-a.service", "svc-b.service", "svc-c.service"}, restartOrder)
}

func TestLifecycle_StartMany_WaitsForUnitGeneration(t *testing.T) {
	// This test verifies that StartMany waits for units to be generated
	// before attempting to start them
	logger := testutil.NewTestLogger(t)

	mockFactory := &MockConnectionFactory{failAttempts: 1}
	// Fail first attempt, succeed on 2nd (simulates generator delay)
	mockConn := &MockConnection{
		factory:  mockFactory,
		props:    map[string]interface{}{"LoadState": "loaded"},
		failMode: 2,
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	results := l.StartMany(ctx, []string{"test-svc"})

	// Should succeed after waiting for unit generation
	assert.Len(t, results, 1)
	assert.NoError(t, results["test-svc"])
	// Should have made multiple attempts to verify generation
	assert.Greater(t, mockFactory.attemptCount, 1)
}

func TestLifecycle_StartMany_FailsOnGenerationTimeout(t *testing.T) {
	// This test verifies that StartMany fails appropriately when
	// units fail to be generated within the timeout
	logger := testutil.NewTestLogger(t)

	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{factory: mockFactory, failMode: 1} // Always fail
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(100 * time.Millisecond) // Very short timeout

	ctx := context.Background()
	results := l.StartMany(ctx, []string{"test-svc"})

	// Should fail with generation timeout error
	assert.Len(t, results, 1)
	assert.Error(t, results["test-svc"])
	assert.Contains(t, results["test-svc"].Error(), "failed to reach 'loaded' state")
}

func TestLifecycle_StopMany_WaitsForUnitGeneration(t *testing.T) {
	// This test verifies that StopMany waits for units to be generated
	// before attempting to stop them
	logger := testutil.NewTestLogger(t)

	mockFactory := &MockConnectionFactory{failAttempts: 1}
	// Fail first attempt, succeed on 2nd (simulates generator delay)
	mockConn := &MockConnection{
		factory:  mockFactory,
		props:    map[string]interface{}{"LoadState": "loaded"},
		failMode: 2,
	}
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(1 * time.Second)

	ctx := context.Background()
	results := l.StopMany(ctx, []string{"test-svc"})

	// Should succeed after waiting for unit generation
	assert.Len(t, results, 1)
	assert.NoError(t, results["test-svc"])
	// Should have made multiple attempts to verify generation
	assert.Greater(t, mockFactory.attemptCount, 1)
}

func TestLifecycle_StopMany_FailsOnGenerationTimeout(t *testing.T) {
	// This test verifies that StopMany fails appropriately when
	// units fail to be generated within the timeout
	logger := testutil.NewTestLogger(t)

	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{factory: mockFactory, failMode: 1} // Always fail
	mockFactory.connection = mockConn

	l := NewLifecycle(nil, mockFactory, false, logger)
	l.SetUnitGenerationTimeout(100 * time.Millisecond) // Very short timeout

	ctx := context.Background()
	results := l.StopMany(ctx, []string{"test-svc"})

	// Should fail with generation timeout error
	assert.Len(t, results, 1)
	assert.Error(t, results["test-svc"])
	assert.Contains(t, results["test-svc"].Error(), "failed to reach 'loaded' state")
}
