package cmd

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/testutil"
)

// mockFileInfo implements fs.FileInfo for testing.
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestFileSystemOps_Stat_WithFunc(t *testing.T) {
	expectedInfo := mockFileInfo{name: "test.txt", size: 100}
	expectedErr := errors.New("stat error")

	ops := FileSystemOps{
		StatFunc: func(path string) (fs.FileInfo, error) {
			assert.Equal(t, "/test/path", path)
			return expectedInfo, expectedErr
		},
	}

	info, err := ops.Stat("/test/path")
	assert.Equal(t, expectedInfo, info)
	assert.Equal(t, expectedErr, err)
}

func TestFileSystemOps_Stat_DefaultBehavior(t *testing.T) {
	ops := FileSystemOps{}

	// Test with a real file
	tempFile, err := os.CreateTemp("", "test-stat-*.txt")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()
	_ = tempFile.Close()

	info, err := ops.Stat(tempFile.Name())
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.False(t, info.IsDir())
}

func TestFileSystemOps_WriteFile_WithFunc(t *testing.T) {
	expectedErr := errors.New("write error")
	var capturedPath string
	var capturedData []byte
	var capturedPerm fs.FileMode

	ops := FileSystemOps{
		WriteFileFunc: func(path string, data []byte, perm fs.FileMode) error {
			capturedPath = path
			capturedData = data
			capturedPerm = perm
			return expectedErr
		},
	}

	testData := []byte("test data")
	err := ops.WriteFile("/test/file", testData, 0644)

	assert.Equal(t, "/test/file", capturedPath)
	assert.Equal(t, testData, capturedData)
	assert.Equal(t, fs.FileMode(0644), capturedPerm)
	assert.Equal(t, expectedErr, err)
}

func TestFileSystemOps_WriteFile_DefaultBehavior(t *testing.T) {
	ops := FileSystemOps{}

	tempDir := t.TempDir()
	testFile := tempDir + "/test-write.txt"
	testData := []byte("test content")

	err := ops.WriteFile(testFile, testData, 0600)
	require.NoError(t, err)

	// Verify file was written
	content, err := os.ReadFile(testFile) // #nosec G304 -- test file path is controlled in test
	require.NoError(t, err)
	assert.Equal(t, testData, content)
}

func TestFileSystemOps_Remove_WithFunc(t *testing.T) {
	expectedErr := errors.New("remove error")
	var capturedPath string

	ops := FileSystemOps{
		RemoveFunc: func(path string) error {
			capturedPath = path
			return expectedErr
		},
	}

	err := ops.Remove("/test/path")
	assert.Equal(t, "/test/path", capturedPath)
	assert.Equal(t, expectedErr, err)
}

func TestFileSystemOps_Remove_DefaultBehavior(t *testing.T) {
	ops := FileSystemOps{}

	tempFile, err := os.CreateTemp("", "test-remove-*.txt")
	require.NoError(t, err)
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	// Verify file exists
	_, err = os.Stat(tempPath)
	require.NoError(t, err)

	// Remove it
	err = ops.Remove(tempPath)
	require.NoError(t, err)

	// Verify file is gone
	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err))
}

func TestFileSystemOps_MkdirAll_WithFunc(t *testing.T) {
	expectedErr := errors.New("mkdir error")
	var capturedPath string
	var capturedPerm fs.FileMode

	ops := FileSystemOps{
		MkdirAllFunc: func(path string, perm fs.FileMode) error {
			capturedPath = path
			capturedPerm = perm
			return expectedErr
		},
	}

	err := ops.MkdirAll("/test/dir", 0750)
	assert.Equal(t, "/test/dir", capturedPath)
	assert.Equal(t, fs.FileMode(0750), capturedPerm)
	assert.Equal(t, expectedErr, err)
}

func TestFileSystemOps_MkdirAll_DefaultBehavior(t *testing.T) {
	ops := FileSystemOps{}

	tempDir := t.TempDir()
	testPath := tempDir + "/nested/dir/structure"

	err := ops.MkdirAll(testPath, 0750)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(testPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewFileSystemOps(t *testing.T) {
	ops := NewFileSystemOps()

	// Verify it's a zero-value struct
	assert.Nil(t, ops.StatFunc)
	assert.Nil(t, ops.WriteFileFunc)
	assert.Nil(t, ops.RemoveFunc)
	assert.Nil(t, ops.MkdirAllFunc)

	// Verify it can still be used with default behavior
	tempDir := t.TempDir()
	err := ops.MkdirAll(tempDir+"/test", 0750)
	assert.NoError(t, err)
}

func TestFileSystemOps_ImplementsInterface(_ *testing.T) {
	var _ FileSystem = (*FileSystemOps)(nil)
	var _ FileSystem = &FileSystemOps{}
}

func TestNewCommonDeps(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	deps := NewCommonDeps(logger)

	assert.NotNil(t, deps.Clock)
	assert.NotNil(t, deps.FileSystem)
	assert.Equal(t, logger, deps.Logger)

	// Verify FileSystem can be used
	ops, ok := deps.FileSystem.(*FileSystemOps)
	assert.True(t, ok)
	assert.NotNil(t, ops)
}
