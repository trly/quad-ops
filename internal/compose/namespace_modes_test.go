package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecConverter_NamespaceModes(t *testing.T) {
	tests := []struct {
		name       string
		pidMode    string
		ipcMode    string
		cgroupMode string
	}{
		{
			name:    "pid host",
			pidMode: "host",
		},
		{
			name:    "pid service reference",
			pidMode: "service:db",
		},
		{
			name:    "pid container reference",
			pidMode: "container:my-container",
		},
		{
			name:    "ipc host",
			ipcMode: "host",
		},
		{
			name:    "ipc shareable",
			ipcMode: "shareable",
		},
		{
			name:    "ipc container reference",
			ipcMode: "container:my-container",
		},
		{
			name:       "cgroup host",
			cgroupMode: "host",
		},
		{
			name:       "cgroup private",
			cgroupMode: "private",
		},
		{
			name:       "all namespace modes",
			pidMode:    "host",
			ipcMode:    "shareable",
			cgroupMode: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSpecConverter("/tmp")

			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": types.ServiceConfig{
						Name:   "web",
						Image:  "nginx:latest",
						Pid:    tt.pidMode,
						Ipc:    tt.ipcMode,
						Cgroup: tt.cgroupMode,
					},
				},
			}

			specs, err := sc.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			spec := specs[0]
			assert.Equal(t, tt.pidMode, spec.Container.PidMode)
			assert.Equal(t, tt.ipcMode, spec.Container.IpcMode)
			assert.Equal(t, tt.cgroupMode, spec.Container.CgroupMode)
		})
	}
}
