package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/state"
	"github.com/trly/quad-ops/internal/systemd"
)

// noopClient is a systemd.Client that does nothing, for unit tests
// that exercise file cleanup without a real D-Bus connection.
type noopClient struct{}

func (noopClient) Start(context.Context, ...string) error   { return nil }
func (noopClient) Stop(context.Context, ...string) error    { return nil }
func (noopClient) Restart(context.Context, ...string) error { return nil }
func (noopClient) Reload(context.Context, ...string) error  { return nil }
func (noopClient) DaemonReload(context.Context) error       { return nil }
func (noopClient) Enable(context.Context, ...string) error  { return nil }
func (noopClient) Disable(context.Context, ...string) error { return nil }
func (noopClient) Close() error                             { return nil }

var _ systemd.Client = noopClient{}

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

	deployState := &state.State{
		Repositories: make(map[string]state.RepoState),
		ManagedUnits: make(map[string][]string),
		UnitStates: map[string]state.UnitState{
			"app-old.container": {ContentHash: "abc"},
		},
	}

	ctx := context.Background()
	sync.cleanupStaleUnits(ctx, globals, deployState, noopClient{}, staleFiles)

	// Verify unit state was cleaned up for stale container
	_, ok := deployState.GetUnitState("app-old.container")
	if ok {
		t.Error("expected unit state for app-old.container to be removed")
	}

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
