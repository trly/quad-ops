package cmd

import (
	"io/fs"
	"os"

	"github.com/benbjohnson/clock"
	"github.com/trly/quad-ops/internal/log"
)

// FileSystem defines the interface for file system operations.
type FileSystem interface {
	Stat(string) (fs.FileInfo, error)
	WriteFile(string, []byte, fs.FileMode) error
	Remove(string) error
	MkdirAll(string, fs.FileMode) error
}

// FileSystemOps provides file system operations for dependency injection.
type FileSystemOps struct {
	// Keep public fields for test compatibility
	StatFunc      func(string) (fs.FileInfo, error)
	WriteFileFunc func(string, []byte, fs.FileMode) error
	RemoveFunc    func(string) error
	MkdirAllFunc  func(string, fs.FileMode) error
}

// Stat returns file information for the given path.
func (f *FileSystemOps) Stat(path string) (fs.FileInfo, error) {
	if f.StatFunc != nil {
		return f.StatFunc(path)
	}
	return os.Stat(path)
}

// WriteFile writes data to the given path with specified permissions.
func (f *FileSystemOps) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if f.WriteFileFunc != nil {
		return f.WriteFileFunc(path, data, perm)
	}
	return os.WriteFile(path, data, perm)
}

// Remove deletes the file at the given path.
func (f *FileSystemOps) Remove(path string) error {
	if f.RemoveFunc != nil {
		return f.RemoveFunc(path)
	}
	return os.Remove(path)
}

// MkdirAll creates directories with the given path and permissions.
func (f *FileSystemOps) MkdirAll(path string, perm fs.FileMode) error {
	if f.MkdirAllFunc != nil {
		return f.MkdirAllFunc(path, perm)
	}
	return os.MkdirAll(path, perm)
}

// Ensure FileSystemOps implements FileSystem.
var _ FileSystem = (*FileSystemOps)(nil)

// NewFileSystemOps returns production file system operations.
func NewFileSystemOps() FileSystemOps {
	// Return empty struct - methods will use OS functions as defaults
	return FileSystemOps{}
}

// NotifyFunc represents systemd notification function.
type NotifyFunc func(unsetEnvironment bool, state string) (bool, error)

// CommonDeps provides dependencies common across commands.
type CommonDeps struct {
	Clock      clock.Clock
	FileSystem FileSystem
	Logger     log.Logger
}

// NewCommonDeps creates production common dependencies.
func NewCommonDeps(logger log.Logger) CommonDeps {
	fs := NewFileSystemOps()
	return CommonDeps{
		Clock:      clock.New(),
		FileSystem: &fs,
		Logger:     logger,
	}
}

// NewRootDeps creates common root dependencies for all commands.
// This helper reduces duplication in buildDeps methods.
func NewRootDeps(app *App) CommonDeps {
	return NewCommonDeps(app.Logger)
}
