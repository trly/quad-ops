package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/testutil/fakerunner"
)

func TestVerifySystemRequirements_Success(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates all commands succeeding
	runner := fakerunner.New()
	runner.SetOutput("systemctl", []string{"--version"}, []byte("systemd 247 (247.3-7+deb11u4)"))
	runner.SetOutput("podman", []string{"--version"}, []byte("podman version 3.4.4"))
	runner.SetOutput("test", []string{"-f", "/usr/lib/systemd/system-generators/podman-system-generator"}, []byte(""))

	// Create validator with mock runner
	validator := NewValidator(logger, runner)

	// Run test
	err := validator.SystemRequirements()
	assert.NoError(t, err)
}

func TestVerifySystemRequirements_MissingSystemd(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates systemd missing
	runner := fakerunner.New()
	runner.SetError("systemctl", []string{"--version"}, assert.AnError)

	// Create validator with mock runner
	validator := NewValidator(logger, runner)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not found")
}

func TestVerifySystemRequirements_InvalidSystemd(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates invalid systemd output
	runner := fakerunner.New()
	runner.SetOutput("systemctl", []string{"--version"}, []byte("Something completely different without the expected string"))
	runner.SetOutput("podman", []string{"--version"}, []byte("podman version 3.4.4"))
	runner.SetOutput("test", []string{"-f", "/usr/lib/systemd/system-generators/podman-system-generator"}, []byte(""))

	// Create validator with mock runner
	validator := NewValidator(logger, runner)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemd not properly installed")
}

func TestVerifySystemRequirements_MissingPodman(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates podman missing
	runner := fakerunner.New()
	runner.SetOutput("systemctl", []string{"--version"}, []byte("systemd 247 (247.3-7+deb11u4)"))
	runner.SetError("podman", []string{"--version"}, assert.AnError)

	// Create validator with mock runner
	validator := NewValidator(logger, runner)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "podman not found")
}

func TestVerifySystemRequirements_MissingPodmanGenerator(t *testing.T) {
	// Create logger for testing
	logger := log.NewLogger(true)

	// Create mock runner that simulates podman-system-generator missing
	runner := fakerunner.New()
	runner.SetOutput("systemctl", []string{"--version"}, []byte("systemd 247 (247.3-7+deb11u4)"))
	runner.SetOutput("podman", []string{"--version"}, []byte("podman version 3.4.4"))
	runner.SetError("test", []string{"-f", "/usr/lib/systemd/system-generators/podman-system-generator"}, assert.AnError)

	// Create validator with mock runner
	validator := NewValidator(logger, runner)

	// Run test and check for expected error
	err := validator.SystemRequirements()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "podman systemd generator not found")
}
