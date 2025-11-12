package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestRenderer_Name(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)
	assert.Equal(t, "systemd", r.Name())
}
