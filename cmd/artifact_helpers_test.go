package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
)

func TestFilterLaunchdArtifacts(t *testing.T) {
	tests := []struct {
		name        string
		artifacts   []platform.Artifact
		labelPrefix string
		expected    int
		note        string
	}{
		{
			name: "filters artifacts with matching prefix",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.plist", Hash: "abc123"},
				{Path: "dev.trly.quad-ops.db.plist", Hash: "def456"},
				{Path: "com.other.service.plist", Hash: "xyz789"},
			},
			labelPrefix: "dev.trly.quad-ops",
			expected:    2,
			note:        "Should only return artifacts with dev.trly.quad-ops prefix",
		},
		{
			name: "ignores non-plist files",
			artifacts: []platform.Artifact{
				{Path: "dev.trly.quad-ops.web.plist", Hash: "abc123"},
				{Path: "dev.trly.quad-ops.config.txt", Hash: "txt123"},
				{Path: "dev.trly.quad-ops.readme.md", Hash: "md456"},
			},
			labelPrefix: "dev.trly.quad-ops",
			expected:    1,
			note:        "Should only return .plist files",
		},
		{
			name: "returns empty when no matches",
			artifacts: []platform.Artifact{
				{Path: "com.other.service.plist", Hash: "xyz789"},
				{Path: "io.different.app.plist", Hash: "abc999"},
			},
			labelPrefix: "dev.trly.quad-ops",
			expected:    0,
			note:        "Should return empty list when no artifacts match prefix",
		},
		{
			name:        "handles empty input",
			artifacts:   []platform.Artifact{},
			labelPrefix: "dev.trly.quad-ops",
			expected:    0,
			note:        "Should handle empty artifact list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLaunchdArtifacts(tt.artifacts, tt.labelPrefix)
			assert.Len(t, result, tt.expected, tt.note)
		})
	}
}

func TestFilterQuadletArtifacts(t *testing.T) {
	tests := []struct {
		name      string
		artifacts []platform.Artifact
		expected  int
		note      string
	}{
		{
			name: "filters valid quadlet extensions",
			artifacts: []platform.Artifact{
				{Path: "web.container", Hash: "abc123"},
				{Path: "db.service", Hash: "def456"},
				{Path: "network.network", Hash: "ghi789"},
				{Path: "data.volume", Hash: "jkl012"},
			},
			expected: 4,
			note:     "Should return all valid quadlet artifacts",
		},
		{
			name: "ignores invalid extensions",
			artifacts: []platform.Artifact{
				{Path: "web.container", Hash: "abc123"},
				{Path: ".git/config", Hash: "git123"},
				{Path: "docker-compose.yml", Hash: "yml456"},
				{Path: "README.md", Hash: "md789"},
				{Path: "script.sh", Hash: "sh012"},
			},
			expected: 1,
			note:     "Should only return .container file",
		},
		{
			name: "handles all quadlet extensions",
			artifacts: []platform.Artifact{
				{Path: "app.container", Hash: "c1"},
				{Path: "svc.service", Hash: "s1"},
				{Path: "net.network", Hash: "n1"},
				{Path: "vol.volume", Hash: "v1"},
				{Path: "tgt.target", Hash: "t1"},
				{Path: "tmr.timer", Hash: "tm1"},
				{Path: "bld.build", Hash: "b1"},
			},
			expected: 7,
			note:     "Should accept all valid quadlet extensions",
		},
		{
			name:      "handles empty input",
			artifacts: []platform.Artifact{},
			expected:  0,
			note:      "Should handle empty artifact list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterQuadletArtifacts(tt.artifacts)
			assert.Len(t, result, tt.expected, tt.note)
		})
	}
}

func TestFilterArtifactsForPlatform(t *testing.T) {
	t.Run("filters based on platform", func(t *testing.T) {
		artifacts := []platform.Artifact{
			{Path: "dev.trly.quad-ops.web.plist", Hash: "plist1"},
			{Path: "com.other.service.plist", Hash: "plist2"},
			{Path: "web.container", Hash: "container1"},
			{Path: ".git/config", Hash: "git1"},
		}

		cfg := &config.Settings{
			RepositoryDir: "/repo",
			QuadletDir:    "/quadlet",
			UserMode:      true,
		}

		result := filterArtifactsForPlatform(artifacts, cfg)

		// Result depends on platform - just verify it runs and returns artifacts
		assert.NotNil(t, result)
	})
}

func TestMatchesServiceName(t *testing.T) {
	tests := []struct {
		name         string
		artifactPath string
		serviceName  string
		expected     bool
	}{
		// Systemd direct matches
		{"systemd container", "web-service.container", "web-service", true},
		{"systemd network", "app-network.network", "app-network", true},
		{"systemd volume", "data-volume.volume", "data-volume", true},
		{"systemd no match", "web-service.container", "db-service", false},

		// Launchd suffix matches
		{"launchd suffix match", "com.example.web-service.plist", "web-service", true},
		{"launchd direct match", "web-service.plist", "web-service", true},
		{"launchd no match", "com.example.web-service.plist", "db-service", false},

		// Nested paths
		{"nested systemd", "subdir/web-service.container", "web-service", true},
		{"nested launchd", "path/to/com.example.svc.plist", "svc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesServiceName(tt.artifactPath, tt.serviceName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
