package systemd

import (
	"fmt"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/buildinfo"
)

func TestBaseLabels_AllFieldsPopulated(t *testing.T) {
	repo := RepositoryMeta{
		Name:       "myrepo",
		URL:        "https://github.com/example/repo",
		Ref:        "main",
		ComposeDir: "deploy",
	}
	labels := baseLabels(repo)

	assert.Contains(t, labels, fmt.Sprintf("com.github.trly.quad-ops.version=%s", buildinfo.Version))
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.name=myrepo")
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.url=https://github.com/example/repo")
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.ref=main")
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.compose-dir=deploy")
}

func TestBaseLabels_OptionalFieldsOmitted(t *testing.T) {
	repo := RepositoryMeta{
		Name: "myrepo",
		URL:  "https://github.com/example/repo",
	}
	labels := baseLabels(repo)

	assert.Contains(t, labels, fmt.Sprintf("com.github.trly.quad-ops.version=%s", buildinfo.Version))
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.name=myrepo")
	assert.Contains(t, labels, "com.github.trly.quad-ops.repository.url=https://github.com/example/repo")
	for _, l := range labels {
		assert.NotContains(t, l, "repository.ref=")
		assert.NotContains(t, l, "repository.compose-dir=")
	}
}

func TestBuildContainer_BaseLabelsApplied(t *testing.T) {
	repo := RepositoryMeta{
		Name: "infra",
		URL:  "https://github.com/org/infra",
		Ref:  "v1.0",
	}
	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Labels: types.Labels{
			"app": "web",
		},
	}
	unit := BuildContainer("proj", "web", svc, nil, nil, repo)

	vals := getValues(unit, "Label")
	assert.Contains(t, vals, "app=web")
	assert.Contains(t, vals, fmt.Sprintf("com.github.trly.quad-ops.version=%s", buildinfo.Version))
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.name=infra")
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.url=https://github.com/org/infra")
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.ref=v1.0")
}

func TestBuildVolume_BaseLabelsApplied(t *testing.T) {
	repo := RepositoryMeta{
		Name:       "infra",
		URL:        "https://github.com/org/infra",
		ComposeDir: "stacks",
	}
	vol := &types.VolumeConfig{
		Labels: types.Labels{
			"backup": "true",
		},
	}
	unit := BuildVolume("proj", "data", vol, repo)

	vals := getVolValues(unit, "Label")
	assert.Contains(t, vals, "backup=true")
	assert.Contains(t, vals, fmt.Sprintf("com.github.trly.quad-ops.version=%s", buildinfo.Version))
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.name=infra")
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.compose-dir=stacks")
}

func TestBuildNetwork_BaseLabelsApplied(t *testing.T) {
	repo := RepositoryMeta{
		Name: "infra",
		URL:  "https://github.com/org/infra",
	}
	net := &types.NetworkConfig{
		Labels: types.Labels{
			"tier": "frontend",
		},
	}
	unit := BuildNetwork("proj", "frontend", net, repo)

	vals := getNetValues(unit, "Label")
	assert.Contains(t, vals, "tier=frontend")
	assert.Contains(t, vals, fmt.Sprintf("com.github.trly.quad-ops.version=%s", buildinfo.Version))
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.name=infra")
	assert.Contains(t, vals, "com.github.trly.quad-ops.repository.url=https://github.com/org/infra")
}
