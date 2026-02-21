package systemd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScope_String(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeSystem, "system"},
		{ScopeUser, "user"},
		{ScopeAuto, "auto"},
		{Scope(99), "auto"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.scope.String())
	}
}

func TestError_Error(t *testing.T) {
	t.Run("with unit", func(t *testing.T) {
		err := &Error{Op: "start", Unit: "foo.service", Scope: ScopeUser, Err: errors.New("failed")}
		assert.Equal(t, "systemd start foo.service (user): failed", err.Error())
	})

	t.Run("without unit", func(t *testing.T) {
		err := &Error{Op: "daemon-reload", Scope: ScopeSystem, Err: errors.New("denied")}
		assert.Equal(t, "systemd daemon-reload (system): denied", err.Error())
	})
}

func TestError_Unwrap(t *testing.T) {
	inner := errors.New("connection refused")
	err := &Error{Op: "connect", Scope: ScopeUser, Err: inner}
	assert.Equal(t, inner, err.Unwrap())
	assert.True(t, errors.Is(err, inner))
}

func TestClient_Close_nilConn(t *testing.T) {
	c := &client{conn: nil, scope: ScopeUser}
	assert.NoError(t, c.Close())
}

func TestClient_waitJob(t *testing.T) {
	c := &client{scope: ScopeUser}

	t.Run("done result", func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "done"
		err := c.waitJob(context.Background(), "start", "foo.service", ch)
		assert.NoError(t, err)
	})

	t.Run("failed result", func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "failed"
		err := c.waitJob(context.Background(), "start", "foo.service", ch)
		assert.Error(t, err)

		var sErr *Error
		assert.True(t, errors.As(err, &sErr))
		assert.Equal(t, "start", sErr.Op)
		assert.Equal(t, "foo.service", sErr.Unit)
		assert.Equal(t, ScopeUser, sErr.Scope)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch := make(chan string)
		err := c.waitJob(ctx, "stop", "bar.service", ch)
		assert.Error(t, err)

		var sErr *Error
		assert.True(t, errors.As(err, &sErr))
		assert.Equal(t, "stop", sErr.Op)
		assert.ErrorIs(t, sErr.Err, context.Canceled)
	})
}
