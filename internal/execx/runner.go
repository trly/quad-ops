// Package execx provides a testable abstraction for command execution.
package execx

import (
	"context"
	"os/exec"
)

// Runner defines an interface for executing external commands.
type Runner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealRunner implements Runner using os/exec.
type RealRunner struct{}

// NewRealRunner creates a new RealRunner.
func NewRealRunner() *RealRunner {
	return &RealRunner{}
}

// CombinedOutput executes a command and returns its combined stdout and stderr output.
func (r *RealRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}
