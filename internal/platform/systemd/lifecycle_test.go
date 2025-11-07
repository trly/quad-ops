// Package systemd provides systemd-specific platform implementations.
package systemd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
