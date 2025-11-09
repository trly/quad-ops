// Package systemd provides systemd-specific platform implementations.
package systemd

import (
	"context"
	"errors"
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

func TestLifecycle_waitForUnitGeneration_Success(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockFactory := &MockConnectionFactory{}
	mockConn := &MockConnection{factory: mockFactory, props: map[string]interface{}{}, failMode: 0}
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
	mockConn := &MockConnection{factory: mockFactory, props: map[string]interface{}{}, failMode: 2}
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
	assert.Contains(t, err.Error(), "failed to be generated")
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
