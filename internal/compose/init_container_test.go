package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/unit"
)

func TestCreateInitQuadletUnit(t *testing.T) {
	container := unit.NewContainer("test-init-0")
	container.Image = "alpine:latest"
	container.Exec = []string{"echo", "hello"}

	result := createInitQuadletUnit("test-init-0", container)

	assert.Equal(t, "test-init-0", result.Name)
	assert.Equal(t, "container", result.Type)
	assert.Equal(t, "alpine:latest", result.Container.Image)
	assert.Equal(t, []string{"echo", "hello"}, result.Container.Exec)

	// Check systemd configuration for init containers
	assert.Equal(t, "oneshot", result.Systemd.Type)
	assert.True(t, result.Systemd.RemainAfterExit)
}
