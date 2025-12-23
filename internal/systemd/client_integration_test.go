//go:build integration

package systemd

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func skipIfNoSystemd(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		t.Skip("systemd not available")
	}
}

func TestNew_SystemScope(t *testing.T) {
	skipIfNoSystemd(t)

	if os.Getuid() != 0 {
		t.Skip("system scope requires root")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := New(ctx, ScopeSystem)
	require.NoError(t, err)
	defer c.Close()
}

func TestNew_UserScope(t *testing.T) {
	skipIfNoSystemd(t)

	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("user session bus not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := New(ctx, ScopeUser)
	require.NoError(t, err)
	defer c.Close()
}

func TestDaemonReload(t *testing.T) {
	skipIfNoSystemd(t)

	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("user session bus not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := New(ctx, ScopeUser)
	require.NoError(t, err)
	defer c.Close()

	err = c.DaemonReload(ctx)
	require.NoError(t, err)
}
