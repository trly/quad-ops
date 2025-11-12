package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestFileArtifactStore_Write(t *testing.T) {
	tests := []struct {
		name          string
		artifacts     []platform.Artifact
		expectedPaths []string
		wantErr       bool
	}{
		{
			name: "write new artifacts",
			artifacts: []platform.Artifact{
				{
					Path:    "test.container",
					Content: []byte("test content"),
					Mode:    0644,
					Hash:    "d8e8fca2dc0f896fd7cb4cb0031ba249",
				},
			},
			expectedPaths: []string{"test.container"},
			wantErr:       false,
		},
		{
			name: "write multiple artifacts",
			artifacts: []platform.Artifact{
				{
					Path:    "app.container",
					Content: []byte("container content"),
					Mode:    0644,
				},
				{
					Path:    "app.network",
					Content: []byte("network content"),
					Mode:    0644,
				},
			},
			expectedPaths: []string{"app.container", "app.network"},
			wantErr:       false,
		},
		{
			name: "write to subdirectory",
			artifacts: []platform.Artifact{
				{
					Path:    "subdir/test.container",
					Content: []byte("nested content"),
					Mode:    0600,
				},
			},
			expectedPaths: []string{"subdir/test.container"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tempDir := t.TempDir()

			// Create dependencies
			logger := testutil.NewTestLogger(t)
			mockConfig := testutil.NewMockConfig(t)
			mockConfig.GetConfig().QuadletDir = tempDir
			fsService := fs.NewServiceWithLogger(mockConfig, logger)

			// Create artifact store
			store := NewArtifactStore(fsService, logger, tempDir)

			// Write artifacts
			changedPaths, err := store.Write(context.Background(), tt.artifacts)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if diff := cmp.Diff(tt.expectedPaths, changedPaths, cmpopts.SortSlices(func(a, b string) bool {
				return a < b
			})); diff != "" {
				t.Errorf("changed paths mismatch (-want +got):\n%s", diff)
			}

			// Verify files exist with correct content
			for _, artifact := range tt.artifacts {
				targetPath := filepath.Join(tempDir, artifact.Path)
				require.FileExists(t, targetPath)

				content, err := os.ReadFile(targetPath) //nolint:gosec // Safe as path is constructed in test
				require.NoError(t, err)
				assert.Equal(t, artifact.Content, content)

				// Verify file permissions if specified
				if artifact.Mode != 0 {
					info, err := os.Stat(targetPath)
					require.NoError(t, err)
					assert.Equal(t, artifact.Mode, info.Mode().Perm())
				}
			}
		})
	}
}

func TestFileArtifactStore_Write_ChangeDetection(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	artifact := platform.Artifact{
		Path:    "test.container",
		Content: []byte("initial content"),
		Mode:    0644,
	}

	// First write should detect change
	changedPaths, err := store.Write(context.Background(), []platform.Artifact{artifact})
	require.NoError(t, err)
	assert.Equal(t, []string{"test.container"}, changedPaths)

	// Second write with same content should detect no change
	changedPaths, err = store.Write(context.Background(), []platform.Artifact{artifact})
	require.NoError(t, err)
	assert.Empty(t, changedPaths, "should detect no changes when content is identical")

	// Write with different content should detect change
	artifact.Content = []byte("updated content")
	changedPaths, err = store.Write(context.Background(), []platform.Artifact{artifact})
	require.NoError(t, err)
	assert.Equal(t, []string{"test.container"}, changedPaths)
}

func TestFileArtifactStore_Write_Atomic(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	artifact := platform.Artifact{
		Path:    "test.container",
		Content: []byte("test content"),
		Mode:    0600,
	}

	// Write artifact
	_, err := store.Write(context.Background(), []platform.Artifact{artifact})
	require.NoError(t, err)

	// Verify no temp files left behind
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp", "should not leave temp files")
		assert.NotContains(t, entry.Name(), ".artifact-", "should not leave temp files")
	}

	// Verify final file exists
	targetPath := filepath.Join(tempDir, artifact.Path)
	require.FileExists(t, targetPath)
}

func TestFileArtifactStore_List(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	// Write some artifacts
	artifacts := []platform.Artifact{
		{Path: "app.container", Content: []byte("container"), Mode: 0644},
		{Path: "app.network", Content: []byte("network"), Mode: 0644},
		{Path: "subdir/db.volume", Content: []byte("volume"), Mode: 0600},
	}

	_, err := store.Write(context.Background(), artifacts)
	require.NoError(t, err)

	// List artifacts
	listed, err := store.List(context.Background())
	require.NoError(t, err)

	// Verify count
	assert.Len(t, listed, 3)

	// Verify paths are present (order may vary)
	paths := make([]string, len(listed))
	for i, a := range listed {
		paths[i] = a.Path
	}
	want := []string{"app.container", "app.network", "subdir/db.volume"}
	if diff := cmp.Diff(want, paths, cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	})); diff != "" {
		t.Errorf("artifact paths mismatch (-want +got):\n%s", diff)
	}

	// Verify content is correct
	for _, artifact := range listed {
		var expected []byte
		for _, orig := range artifacts {
			if orig.Path == artifact.Path {
				expected = orig.Content
				break
			}
		}
		assert.Equal(t, expected, artifact.Content)
		assert.NotEmpty(t, artifact.Hash, "hash should be calculated")
	}
}

func TestFileArtifactStore_List_EmptyDirectory(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	// List should return empty slice, not error
	listed, err := store.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestFileArtifactStore_Delete(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	// Write artifacts
	artifacts := []platform.Artifact{
		{Path: "app.container", Content: []byte("container"), Mode: 0644},
		{Path: "app.network", Content: []byte("network"), Mode: 0644},
	}

	_, err := store.Write(context.Background(), artifacts)
	require.NoError(t, err)

	// Delete one artifact
	err = store.Delete(context.Background(), []string{"app.container"})
	require.NoError(t, err)

	// Verify it's deleted
	targetPath := filepath.Join(tempDir, "app.container")
	assert.NoFileExists(t, targetPath)

	// Verify other artifact still exists
	otherPath := filepath.Join(tempDir, "app.network")
	assert.FileExists(t, otherPath)
}

func TestFileArtifactStore_Delete_NonExistent(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	// Delete non-existent file should not error
	err := store.Delete(context.Background(), []string{"nonexistent.container"})
	require.NoError(t, err, "deleting non-existent file should not error")
}

func TestFileArtifactStore_ContextCancellation(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create dependencies
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)
	mockConfig.GetConfig().QuadletDir = tempDir
	fsService := fs.NewServiceWithLogger(mockConfig, logger)
	store := NewArtifactStore(fsService, logger, tempDir)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Write should respect context
	artifacts := []platform.Artifact{
		{Path: "test.container", Content: []byte("test"), Mode: 0644},
	}
	_, err := store.Write(ctx, artifacts)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// List should respect context
	_, err = store.List(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// Delete should respect context
	err = store.Delete(ctx, []string{"test.container"})
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
