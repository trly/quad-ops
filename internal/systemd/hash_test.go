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
	workingDir := t.TempDir()
	svc := &types.ServiceConfig{Image: "nginx:latest"}

	us := ComputeUnitState(unit, svc, workingDir, workingDir)
	assert.NotEmpty(t, us.ContentHash)
	assert.Empty(t, us.BindMountHashes)

	// Same content produces same hash
	us2 := ComputeUnitState(unit, svc, workingDir, workingDir)
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
	workingDir := t.TempDir()
	svc := &types.ServiceConfig{Image: "nginx:latest"}

	us1 := ComputeUnitState(unit1, svc, workingDir, workingDir)
	us2 := ComputeUnitState(unit2, svc, workingDir, workingDir)
	assert.NotEqual(t, us1.ContentHash, us2.ContentHash)
}

func TestCollectBindMountHashesInProjectDir(t *testing.T) {
	repoDir := t.TempDir()
	confFile := filepath.Join(repoDir, "nginx.conf")
	require.NoError(t, os.WriteFile(confFile, []byte("server {}"), 0o644))

	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: "nginx.conf", Target: "/etc/nginx/nginx.conf"},
		},
	}

	hashes := CollectBindMountHashes(svc, repoDir, repoDir)
	assert.Len(t, hashes, 1)
	assert.Contains(t, hashes, confFile)
	assert.NotEmpty(t, hashes[confFile])
}

func TestCollectBindMountHashesSkipsExternalPaths(t *testing.T) {
	repoDir := t.TempDir()
	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: "/etc/timezone", Target: "/etc/timezone"},
		},
	}

	hashes := CollectBindMountHashes(svc, repoDir, repoDir)
	assert.Empty(t, hashes)
}

func TestCollectBindMountHashesSkipsDirectories(t *testing.T) {
	repoDir := t.TempDir()
	subDir := filepath.Join(repoDir, "config")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: subDir, Target: "/config"},
		},
	}

	hashes := CollectBindMountHashes(svc, repoDir, repoDir)
	assert.Empty(t, hashes)
}

func TestCollectBindMountHashesSkipsNamedVolumes(t *testing.T) {
	repoDir := t.TempDir()
	svc := &types.ServiceConfig{
		Image: "postgres:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeVolume, Source: "pgdata", Target: "/var/lib/postgresql/data"},
		},
	}

	hashes := CollectBindMountHashes(svc, repoDir, repoDir)
	assert.Empty(t, hashes)
}

func TestCollectBindMountHashesPerServiceIsolation(t *testing.T) {
	repoDir := t.TempDir()
	confA := filepath.Join(repoDir, "config-a.txt")
	confB := filepath.Join(repoDir, "config-b.txt")
	require.NoError(t, os.WriteFile(confA, []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(confB, []byte("b"), 0o644))

	svcA := &types.ServiceConfig{
		Image: "app-a:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: confA, Target: "/config"},
		},
	}
	svcB := &types.ServiceConfig{
		Image: "app-b:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: confB, Target: "/config"},
		},
	}

	hashesA := CollectBindMountHashes(svcA, repoDir, repoDir)
	hashesB := CollectBindMountHashes(svcB, repoDir, repoDir)

	assert.Len(t, hashesA, 1, "service A should only hash its own bind mount")
	assert.Len(t, hashesB, 1, "service B should only hash its own bind mount")
	assert.Contains(t, hashesA, confA)
	assert.NotContains(t, hashesA, confB)
	assert.Contains(t, hashesB, confB)
	assert.NotContains(t, hashesB, confA)
}
