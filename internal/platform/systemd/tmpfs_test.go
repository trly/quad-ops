package systemd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestRenderer_TmpfsOptions(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "test",
		Description: "Test tmpfs options",
		Container: service.Container{
			Image: "nginx:alpine",
			Mounts: []service.Mount{
				{
					Target:   "/tmp/data",
					Type:     service.MountTypeTmpfs,
					ReadOnly: false,
					TmpfsOptions: &service.TmpfsOptions{
						Size: "64m",
						Mode: 1777,
						UID:  1000,
						GID:  1000,
					},
				},
			},
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Len(t, result.Artifacts, 1)
	require.Equal(t, "test.container", result.Artifacts[0].Path)

	content := string(result.Artifacts[0].Content)
	t.Logf("Generated content:\n%s", content)

	assert.Contains(t, content, "Tmpfs=/tmp/data:rw,size=64m,mode=1777,uid=1000,gid=1000")
}
