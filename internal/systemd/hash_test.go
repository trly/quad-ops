package systemd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeUnitStateContentHash(t *testing.T) {
	unit := Unit{
		Name: "app-web.container",
		File: testIniFile("Container", map[string]string{"Image": "nginx:latest"}),
	}
	project := &types.Project{
		Name:       "app",
		WorkingDir: t.TempDir(),
	}

	us := ComputeUnitState(unit, project, project.WorkingDir)
	assert.NotEmpty(t, us.ContentHash)
	assert.Empty(t, us.BindMountHashes)

	// Same content produces same hash
	us2 := ComputeUnitState(unit, project, project.WorkingDir)
	assert.Equal(t, us.ContentHash, us2.ContentHash)
}

func TestComputeUnitStateDifferentContentDifferentHash(t *testing.T) {
	unit1 := Unit{
		Name: "app-web.container",
		File: testIniFile("Container", map[string]string{"Image": "nginx:1.0"}),
	}
	unit2 := Unit{
		Name: "app-web.container",
		File: testIniFile("Container", map[string]string{"Image": "nginx:2.0"}),
	}
	project := &types.Project{
		Name:       "app",
		WorkingDir: t.TempDir(),
	}

	us1 := ComputeUnitState(unit1, project, project.WorkingDir)
	us2 := ComputeUnitState(unit2, project, project.WorkingDir)
	assert.NotEqual(t, us1.ContentHash, us2.ContentHash)
}

func TestCollectBindMountHashesInProjectDir(t *testing.T) {
	repoDir := t.TempDir()
	confFile := filepath.Join(repoDir, "nginx.conf")
	require.NoError(t, os.WriteFile(confFile, []byte("server {}"), 0o644))

	project := &types.Project{
		Name:       "app",
		WorkingDir: repoDir,
		Services: types.Services{
			"web": {
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeBind, Source: "nginx.conf", Target: "/etc/nginx/nginx.conf"},
				},
			},
		},
	}

	hashes := CollectBindMountHashes(project, repoDir)
	assert.Len(t, hashes, 1)
	assert.Contains(t, hashes, confFile)
	assert.NotEmpty(t, hashes[confFile])
}

func TestCollectBindMountHashesSkipsExternalPaths(t *testing.T) {
	repoDir := t.TempDir()
	project := &types.Project{
		Name:       "app",
		WorkingDir: repoDir,
		Services: types.Services{
			"web": {
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeBind, Source: "/etc/timezone", Target: "/etc/timezone"},
				},
			},
		},
	}

	hashes := CollectBindMountHashes(project, repoDir)
	assert.Empty(t, hashes)
}

func TestCollectBindMountHashesSkipsDirectories(t *testing.T) {
	repoDir := t.TempDir()
	subDir := filepath.Join(repoDir, "config")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	project := &types.Project{
		Name:       "app",
		WorkingDir: repoDir,
		Services: types.Services{
			"web": {
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeBind, Source: subDir, Target: "/config"},
				},
			},
		},
	}

	hashes := CollectBindMountHashes(project, repoDir)
	assert.Empty(t, hashes)
}

func TestCollectBindMountHashesSkipsNamedVolumes(t *testing.T) {
	repoDir := t.TempDir()
	project := &types.Project{
		Name:       "app",
		WorkingDir: repoDir,
		Services: types.Services{
			"db": {
				Image: "postgres:latest",
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeVolume, Source: "pgdata", Target: "/var/lib/postgresql/data"},
				},
			},
		},
	}

	hashes := CollectBindMountHashes(project, repoDir)
	assert.Empty(t, hashes)
}
