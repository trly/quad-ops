// Package state manages deployment state for quad-ops, tracking
// current and previous commit hashes per repository to enable rollback.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RepoState tracks the deployed commit hashes for a single repository.
type RepoState struct {
	Current  string `json:"current"`
	Previous string `json:"previous,omitempty"`
}

// State holds the deployment state for all repositories.
type State struct {
	Repositories map[string]RepoState `json:"repositories"`
	ManagedUnits map[string][]string  `json:"managed_units,omitempty"`
}

// Load reads the state file from disk. Returns an empty state if the file does not exist.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				Repositories: make(map[string]RepoState),
				ManagedUnits: make(map[string][]string),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	s := &State{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if s.Repositories == nil {
		s.Repositories = make(map[string]RepoState)
	}

	if s.ManagedUnits == nil {
		s.ManagedUnits = make(map[string][]string)
	}

	return s, nil
}

// Save writes the state to disk, creating parent directories as needed.
func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// SetCommit records a new deployment for the named repository,
// shifting the current commit to previous.
func (s *State) SetCommit(repoName, commitHash string) {
	rs := s.Repositories[repoName]
	if rs.Current != commitHash {
		rs.Previous = rs.Current
		rs.Current = commitHash
	}
	s.Repositories[repoName] = rs
}

// GetPrevious returns the previous commit hash for the named repository.
// Returns empty string if no previous state exists.
func (s *State) GetPrevious(repoName string) string {
	return s.Repositories[repoName].Previous
}

// SetManagedUnits records the quadlet unit filenames managed for a repository.
func (s *State) SetManagedUnits(repoName string, units []string) {
	if s.ManagedUnits == nil {
		s.ManagedUnits = make(map[string][]string)
	}
	s.ManagedUnits[repoName] = units
}

// GetManagedUnits returns the quadlet unit filenames managed for a repository.
func (s *State) GetManagedUnits(repoName string) []string {
	return s.ManagedUnits[repoName]
}
