package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/state"
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

// TestCleanupStaleUnitsRemovesFiles tests that stale unit files are removed from disk.
func TestCleanupStaleUnitsRemovesFiles(t *testing.T) {
	quadletDir := t.TempDir()

	// Create stale unit files
	staleFiles := []string{"app-old.container", "app-oldnet.network", "app-oldvol.volume"}
	for _, name := range staleFiles {
		if err := os.WriteFile(filepath.Join(quadletDir, name), []byte("[Unit]"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a unit that should NOT be removed
	keepFile := filepath.Join(quadletDir, "app-keep.container")
	if err := os.WriteFile(keepFile, []byte("[Unit]"), 0o644); err != nil {
		t.Fatal(err)
	}

	sync := &SyncCmd{}
	globals := &Globals{
		AppCfg: &config.AppConfig{},
	}
	// Override quadlet dir to our temp dir
	globals.AppCfg.QuadletDir = quadletDir

	ctx := context.Background()
	sync.cleanupStaleUnits(ctx, globals, staleFiles)

	// Stale files should be removed
	for _, name := range staleFiles {
		path := filepath.Join(quadletDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected stale unit %s to be removed", name)
		}
	}

	// Keep file should still exist
	if _, err := os.Stat(keepFile); os.IsNotExist(err) {
		t.Error("expected keep file to still exist")
	}
}

// TestDiffUnits tests the set difference computation.
func TestDiffUnits(t *testing.T) {
	old := map[string]struct{}{
		"app-web.container": {},
		"app-db.container":  {},
		"app-net.network":   {},
	}
	current := map[string]struct{}{
		"app-web.container": {},
	}

	stale := diffUnits(old, current)
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale units, got %d", len(stale))
	}

	staleSet := make(map[string]struct{})
	for _, s := range stale {
		staleSet[s] = struct{}{}
	}
	if _, ok := staleSet["app-db.container"]; !ok {
		t.Error("expected app-db.container in stale units")
	}
	if _, ok := staleSet["app-net.network"]; !ok {
		t.Error("expected app-net.network in stale units")
	}
}

// TestDiffUnitsNoChanges tests that no stale units are returned when sets match.
func TestDiffUnitsNoChanges(t *testing.T) {
	units := map[string]struct{}{
		"app-web.container": {},
	}

	stale := diffUnits(units, units)
	if len(stale) != 0 {
		t.Errorf("expected no stale units, got %d", len(stale))
	}
}

// TestCollectAllManagedUnits tests aggregation across multiple repos.
func TestCollectAllManagedUnits(t *testing.T) {
	s := &state.State{
		Repositories: make(map[string]state.RepoState),
		ManagedUnits: map[string][]string{
			"repo-a": {"a-web.container", "a-net.network"},
			"repo-b": {"b-api.container"},
		},
	}

	result := collectAllManagedUnits(s)
	if len(result) != 3 {
		t.Fatalf("expected 3 units, got %d", len(result))
	}

	for _, expected := range []string{"a-web.container", "a-net.network", "b-api.container"} {
		if _, ok := result[expected]; !ok {
			t.Errorf("expected %s in result", expected)
		}
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
