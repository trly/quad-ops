package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNonExistentFile(t *testing.T) {
	s, err := Load("/nonexistent/path/state.json")
	require.NoError(t, err)
	assert.NotNil(t, s.Repositories)
	assert.Empty(t, s.Repositories)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "state.json")

	s := &State{Repositories: make(map[string]RepoState)}
	s.SetCommit("my-repo", "abc123")

	require.NoError(t, s.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "abc123", loaded.Repositories["my-repo"].Current)
	assert.Empty(t, loaded.Repositories["my-repo"].Previous)
}

func TestSetCommitShiftsPrevious(t *testing.T) {
	s := &State{Repositories: make(map[string]RepoState)}

	s.SetCommit("repo", "first")
	assert.Equal(t, "first", s.Repositories["repo"].Current)
	assert.Empty(t, s.Repositories["repo"].Previous)

	s.SetCommit("repo", "second")
	assert.Equal(t, "second", s.Repositories["repo"].Current)
	assert.Equal(t, "first", s.Repositories["repo"].Previous)

	s.SetCommit("repo", "third")
	assert.Equal(t, "third", s.Repositories["repo"].Current)
	assert.Equal(t, "second", s.Repositories["repo"].Previous)
}

func TestSetCommitIdempotent(t *testing.T) {
	s := &State{Repositories: make(map[string]RepoState)}

	s.SetCommit("repo", "abc")
	s.SetCommit("repo", "abc")
	assert.Equal(t, "abc", s.Repositories["repo"].Current)
	assert.Empty(t, s.Repositories["repo"].Previous)
}

func TestGetPreviousMissingRepo(t *testing.T) {
	s := &State{Repositories: make(map[string]RepoState)}
	assert.Empty(t, s.GetPrevious("nonexistent"))
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0o644))

	_, err := Load(path)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse state file")
}

func TestLoadUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o000))

	_, err := Load(path)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to read state file")
}

func TestLoadNullRepositories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"repositories":null}`), 0o644))

	s, err := Load(path)
	require.NoError(t, err)
	assert.NotNil(t, s.Repositories)
	assert.Empty(t, s.Repositories)
}

func TestSaveToUnwritablePath(t *testing.T) {
	s := &State{Repositories: make(map[string]RepoState)}

	err := s.Save("/proc/nonexistent/subdir/state.json")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to create state directory")
}

func TestSaveToUnwritableFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	s := &State{Repositories: make(map[string]RepoState)}
	err := s.Save(filepath.Join(dir, "state.json"))
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to write state file")
}
