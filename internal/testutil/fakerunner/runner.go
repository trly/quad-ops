// Package fakerunner provides a fake implementation of execx.Runner for testing.
package fakerunner

import (
	"context"
	"fmt"
	"strings"
)

// Runner is a fake implementation of execx.Runner for testing.
type Runner struct {
	outputs map[string][]byte
	errors  map[string]error
	calls   []Call
}

// Call represents a captured command execution call.
type Call struct {
	Name string
	Args []string
}

// New creates a new fake runner.
func New() *Runner {
	return &Runner{
		outputs: make(map[string][]byte),
		errors:  make(map[string]error),
		calls:   []Call{},
	}
}

// SetOutput sets the output for a specific command.
func (r *Runner) SetOutput(name string, args []string, output []byte) {
	key := r.makeKey(name, args)
	r.outputs[key] = output
}

// SetError sets the error for a specific command.
func (r *Runner) SetError(name string, args []string, err error) {
	key := r.makeKey(name, args)
	r.errors[key] = err
}

// CombinedOutput implements execx.Runner.
func (r *Runner) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, Call{Name: name, Args: args})

	key := r.makeKey(name, args)

	if err, exists := r.errors[key]; exists {
		return nil, err
	}

	if output, exists := r.outputs[key]; exists {
		return output, nil
	}

	// Default behavior - return empty output and no error
	return []byte{}, nil
}

// GetCalls returns all captured command calls.
func (r *Runner) GetCalls() []Call {
	return r.calls
}

// Reset clears all stored outputs, errors, and calls.
func (r *Runner) Reset() {
	r.outputs = make(map[string][]byte)
	r.errors = make(map[string]error)
	r.calls = []Call{}
}

func (r *Runner) makeKey(name string, args []string) string {
	return fmt.Sprintf("%s %s", name, strings.Join(args, " "))
}
