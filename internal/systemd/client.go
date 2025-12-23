package systemd

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/config"
)

// Scope represents the systemd connection scope.
type Scope int

const (
	// ScopeAuto automatically selects user or system based on UID.
	ScopeAuto Scope = iota
	// ScopeSystem connects to the system bus (requires root).
	ScopeSystem
	// ScopeUser connects to the user session bus.
	ScopeUser
)

func (s Scope) String() string {
	switch s {
	case ScopeSystem:
		return "system"
	case ScopeUser:
		return "user"
	default:
		return "auto"
	}
}

// Client provides operations for managing systemd units via D-Bus.
type Client interface {
	Start(ctx context.Context, units ...string) error
	Stop(ctx context.Context, units ...string) error
	Restart(ctx context.Context, units ...string) error
	Reload(ctx context.Context, units ...string) error
	DaemonReload(ctx context.Context) error
	Enable(ctx context.Context, units ...string) error
	Disable(ctx context.Context, units ...string) error
	Close() error
}

// Error represents a systemd operation error with context.
type Error struct {
	Op    string
	Unit  string
	Scope Scope
	Err   error
}

func (e *Error) Error() string {
	if e.Unit != "" {
		return fmt.Sprintf("systemd %s %s (%s): %v", e.Op, e.Unit, e.Scope, e.Err)
	}
	return fmt.Sprintf("systemd %s (%s): %v", e.Op, e.Scope, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type client struct {
	conn  *dbus.Conn
	scope Scope
}

// New creates a new systemd Client with the given scope.
// If scope is ScopeAuto, it uses ScopeUser when running as non-root.
func New(ctx context.Context, scope Scope) (Client, error) {
	if scope == ScopeAuto {
		if config.IsUserMode() {
			scope = ScopeUser
		} else {
			scope = ScopeSystem
		}
	}

	var (
		conn *dbus.Conn
		err  error
	)

	switch scope {
	case ScopeUser:
		conn, err = dbus.NewUserConnectionContext(ctx)
	case ScopeSystem:
		conn, err = dbus.NewSystemConnectionContext(ctx)
	default:
		return nil, &Error{Op: "connect", Scope: scope, Err: fmt.Errorf("unknown scope: %v", scope)}
	}

	if err != nil {
		return nil, &Error{Op: "connect", Scope: scope, Err: err}
	}

	return &client{conn: conn, scope: scope}, nil
}

const jobMode = "replace"

func (c *client) waitJob(ctx context.Context, op, unit string, ch <-chan string) error {
	select {
	case <-ctx.Done():
		return &Error{Op: op, Unit: unit, Scope: c.scope, Err: ctx.Err()}
	case result := <-ch:
		if result == "done" {
			return nil
		}
		return &Error{Op: op, Unit: unit, Scope: c.scope, Err: fmt.Errorf("job result: %s", result)}
	}
}

func (c *client) Start(ctx context.Context, units ...string) error {
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	wg.Add(len(units))
	for _, u := range units {
		go func(unit string) {
			defer wg.Done()
			ch := make(chan string, 1)
			if _, err := c.conn.StartUnitContext(ctx, unit, jobMode, ch); err != nil {
				mu.Lock()
				errs = append(errs, &Error{Op: "start", Unit: unit, Scope: c.scope, Err: err})
				mu.Unlock()
				return
			}
			if err := c.waitJob(ctx, "start", unit, ch); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(u)
	}

	wg.Wait()
	return errors.Join(errs...)
}

func (c *client) Stop(ctx context.Context, units ...string) error {
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	wg.Add(len(units))
	for _, u := range units {
		go func(unit string) {
			defer wg.Done()
			ch := make(chan string, 1)
			if _, err := c.conn.StopUnitContext(ctx, unit, jobMode, ch); err != nil {
				mu.Lock()
				errs = append(errs, &Error{Op: "stop", Unit: unit, Scope: c.scope, Err: err})
				mu.Unlock()
				return
			}
			if err := c.waitJob(ctx, "stop", unit, ch); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(u)
	}

	wg.Wait()
	return errors.Join(errs...)
}

func (c *client) Restart(ctx context.Context, units ...string) error {
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	wg.Add(len(units))
	for _, u := range units {
		go func(unit string) {
			defer wg.Done()
			ch := make(chan string, 1)
			if _, err := c.conn.RestartUnitContext(ctx, unit, jobMode, ch); err != nil {
				mu.Lock()
				errs = append(errs, &Error{Op: "restart", Unit: unit, Scope: c.scope, Err: err})
				mu.Unlock()
				return
			}
			if err := c.waitJob(ctx, "restart", unit, ch); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(u)
	}

	wg.Wait()
	return errors.Join(errs...)
}

func (c *client) Reload(ctx context.Context, units ...string) error {
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	wg.Add(len(units))
	for _, u := range units {
		go func(unit string) {
			defer wg.Done()
			ch := make(chan string, 1)
			if _, err := c.conn.ReloadUnitContext(ctx, unit, jobMode, ch); err != nil {
				mu.Lock()
				errs = append(errs, &Error{Op: "reload", Unit: unit, Scope: c.scope, Err: err})
				mu.Unlock()
				return
			}
			if err := c.waitJob(ctx, "reload", unit, ch); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(u)
	}

	wg.Wait()
	return errors.Join(errs...)
}

func (c *client) DaemonReload(ctx context.Context) error {
	if err := c.conn.ReloadContext(ctx); err != nil {
		return &Error{Op: "daemon-reload", Scope: c.scope, Err: err}
	}
	return nil
}

func (c *client) Enable(ctx context.Context, units ...string) error {
	_, _, err := c.conn.EnableUnitFilesContext(ctx, units, false, false)
	if err != nil {
		return &Error{Op: "enable", Scope: c.scope, Err: err}
	}
	return nil
}

func (c *client) Disable(ctx context.Context, units ...string) error {
	_, err := c.conn.DisableUnitFilesContext(ctx, units, false)
	if err != nil {
		return &Error{Op: "disable", Scope: c.scope, Err: err}
	}
	return nil
}

func (c *client) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}
