package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/log"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	// Map of command to output and error
	CommandOutputs map[string]struct {
		Output []byte
		Err    error
	}
}

// Run returns mock output based on command.
func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	// Create a key by joining command and args
	key := name
	for _, arg := range args {
		key += " " + arg
	}

	// Look up the response
	if response, ok := m.CommandOutputs[key]; ok {
		return response.Output, response.Err
	}

	// Default error for unknown commands
	return nil, errors.New("command not mocked: " + key)
}

func TestVerifySystemRequirements_Success(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates all commands succeeding
	mock := &MockCommandRunner{
		CommandOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"systemctl --version": {
				Output: []byte("systemd 247 (247.3-7+deb11u4)"),
				Err:    nil,
			},
			"podman --version": {
				Output: []byte("podman version 3.4.4"),
				Err:    nil,
			},
			"test -f /usr/lib/systemd/system-generators/podman-system-generator": {
				Output: []byte(""),
				Err:    nil,
			},
		},
	}

	// Create validator with mock runner
	validator := NewValidator(logger, mock)

	// Run test
	err := validator.SystemRequirements()
	assert.NoError(t, err)
}

func TestVerifySystemRequirements_MissingSystemd(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates systemd missing
	mock := &MockCommandRunner{
		CommandOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"systemctl --version": {
				Output: nil,
				Err:    errors.New("exec: \"systemctl\": executable file not found in $PATH"),
			},
		},
	}

	// Create validator with mock runner
	validator := NewValidator(logger, mock)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestVerifySystemRequirements_InvalidSystemd(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates invalid systemd output
	mock := &MockCommandRunner{
		CommandOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"systemctl --version": {
				// Ensure the output doesn't contain "systemd" anywhere
				Output: []byte("Something completely different without the expected string"),
				Err:    nil,
			},
			// Include these to prevent "command not mocked" errors if execution continues
			"podman --version": {
				Output: []byte("podman version 3.4.4"),
				Err:    nil,
			},
			"test -f /usr/lib/systemd/system-generators/podman-system-generator": {
				Output: []byte(""),
				Err:    nil,
			},
		},
	}

	// Create validator with mock runner
	validator := NewValidator(logger, mock)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not properly installed")
}

func TestVerifySystemRequirements_MissingPodman(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates podman missing
	mock := &MockCommandRunner{
		CommandOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"systemctl --version": {
				Output: []byte("systemd 247 (247.3-7+deb11u4)"),
				Err:    nil,
			},
			"podman --version": {
				Output: nil,
				Err:    errors.New("exec: \"podman\": executable file not found in $PATH"),
			},
		},
	}

	// Create validator with mock runner
	validator := NewValidator(logger, mock)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "podman not found")
}

func TestVerifySystemRequirements_MissingPodmanGenerator(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates podman-system-generator missing
	mock := &MockCommandRunner{
		CommandOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"systemctl --version": {
				Output: []byte("systemd 247 (247.3-7+deb11u4)"),
				Err:    nil,
			},
			"podman --version": {
				Output: []byte("podman version 3.4.4"),
				Err:    nil,
			},
			"test -f /usr/lib/systemd/system-generators/podman-system-generator": {
				Output: nil,
				Err:    errors.New("test failed"),
			},
		},
	}

	// Create validator with mock runner
	validator := NewValidator(logger, mock)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "podman systemd generator not found")
}
