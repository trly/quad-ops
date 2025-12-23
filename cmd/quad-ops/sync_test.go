package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/systemd"
	"gopkg.in/ini.v1"
)

// TestWriteUnitsToQuadletDir tests that units are written with correct file extensions.
func TestWriteUnitsToQuadletDir(t *testing.T) {
	tmpDir := t.TempDir()
	sync := &SyncCmd{}

	// Create test units with different types
	units := []systemd.Unit{
		{
			Name: "my-container.container",
			File: createTestIniFile("Container", map[string]string{"Image": "alpine:latest"}),
		},
		{
			Name: "my-volume.volume",
			File: createTestIniFile("Volume", map[string]string{"Driver": "local"}),
		},
		{
			Name: "my-network.network",
			File: createTestIniFile("Network", map[string]string{"Driver": "bridge"}),
		},
	}

	err := sync.writeUnits(units, tmpDir)
	if err != nil {
		t.Fatalf("writeUnits failed: %v", err)
	}

	// Verify files were created with correct extensions
	tests := []struct {
		name        string
		expectedExt string
	}{
		{"my-container", ".container"},
		{"my-volume", ".volume"},
		{"my-network", ".network"},
	}

	for _, tt := range tests {
		expectedPath := filepath.Join(tmpDir, tt.name+tt.expectedExt)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", expectedPath)
		}

		// Verify file content starts with a section header
		content, _ := os.ReadFile(expectedPath)
		if len(content) == 0 {
			t.Errorf("file %s is empty", expectedPath)
		}
	}
}

// TestRunWithNoConfig tests that Run returns error when config is not loaded.
func TestRunWithNoConfig(t *testing.T) {
	sync := &SyncCmd{}
	globals := &Globals{AppCfg: nil}

	err := sync.Run(globals)
	if err == nil {
		t.Error("expected error when config is not loaded")
	}
}

// TestRunWithNoRepositories tests that Run returns nil when no repositories configured.
func TestRunWithNoRepositories(t *testing.T) {
	sync := &SyncCmd{}
	globals := &Globals{
		AppCfg: &config.AppConfig{
			Repositories: []struct {
				Name       string `yaml:"name"`
				URL        string `yaml:"url"`
				Ref        string `yaml:"ref,omitempty"`
				ComposeDir string `yaml:"composeDir,omitempty"`
			}{},
		},
	}

	err := sync.Run(globals)
	if err != nil {
		t.Errorf("expected no error with no repositories, got %v", err)
	}
}

// createTestIniFile is a helper to create a test ini.File with a section and keys.
func createTestIniFile(sectionName string, keys map[string]string) *ini.File {
	file := ini.Empty()
	section, _ := file.NewSection(sectionName)
	for k, v := range keys {
		_, _ = section.NewKey(k, v)
	}
	return file
}
