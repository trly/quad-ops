package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCommand_Basic tests validate command.
func TestValidateCommand_Basic(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	output, err := ExecuteCommandWithCapture(t, cmd, []string{})

	require.NoError(t, err)
	if output != "" {
		assert.Contains(t, output, "Validating")
	}
}

// TestValidateCommand_Help tests help output.
func TestValidateCommand_Help(t *testing.T) {
	cmd := NewValidateCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "Validates Docker Compose files")
	assert.Contains(t, output, "quad-ops extensions")
}

// TestValidateCommand_Flags tests command flags.
func TestValidateCommand_Flags(t *testing.T) {
	cmd := NewValidateCommand().GetCobraCommand()

	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag)
	assert.Equal(t, "", repoFlag.DefValue)

	refFlag := cmd.Flags().Lookup("ref")
	require.NotNil(t, refFlag)
	assert.Equal(t, "main", refFlag.DefValue)

	composeDirFlag := cmd.Flags().Lookup("compose-dir")
	require.NotNil(t, composeDirFlag)
	assert.Equal(t, "", composeDirFlag.DefValue)

	skipCloneFlag := cmd.Flags().Lookup("skip-clone")
	require.NotNil(t, skipCloneFlag)
	assert.Equal(t, "false", skipCloneFlag.DefValue)

	checkSystemFlag := cmd.Flags().Lookup("check-system")
	require.NotNil(t, checkSystemFlag)
	assert.Equal(t, "false", checkSystemFlag.DefValue)
}

// TestValidateCommand_ValidateDirectory tests validating a directory.
func TestValidateCommand_ValidateDirectory(t *testing.T) {
	tempDir := t.TempDir()

	composeContent := `
version: "3.8"
services:
  test:
    image: nginx:latest
`
	err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeContent), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{tempDir})
	assert.NoError(t, err)
}

// TestValidateCommand_ValidateSingleFile tests validating a single compose file.
func TestValidateCommand_ValidateSingleFile(t *testing.T) {
	tempDir := t.TempDir()
	composeFile := filepath.Join(tempDir, "docker-compose.yml")

	composeContent := `
version: "3.8"
services:
  test:
    image: nginx:latest
`
	err := os.WriteFile(composeFile, []byte(composeContent), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{composeFile})
	assert.NoError(t, err)
}

// TestValidateCommand_InvalidComposeFile tests invalid compose file.
func TestValidateCommand_InvalidComposeFile(t *testing.T) {
	tempDir := t.TempDir()
	composeFile := filepath.Join(tempDir, "invalid.yml")

	err := os.WriteFile(composeFile, []byte("invalid: yaml: content:"), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{composeFile})
	assert.Error(t, err)
}

// TestValidateCommand_NonExistentPath tests non-existent path.
func TestValidateCommand_NonExistentPath(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{"/nonexistent/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestValidateCommand_NonYAMLFile tests non-YAML file.
func TestValidateCommand_NonYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	textFile := filepath.Join(tempDir, "file.txt")

	err := os.WriteFile(textFile, []byte("not a compose file"), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{textFile})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not appear to be a Docker Compose file")
}

// TestValidateCommand_EmptyDirectory tests directory with no compose files.
func TestValidateCommand_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err := ExecuteCommand(t, cmd, []string{tempDir})
	assert.NoError(t, err)
}

// TestValidateCommand_MutuallyExclusiveFlags tests repo flag and path argument exclusivity.
func TestValidateCommand_MutuallyExclusiveFlags(t *testing.T) {
	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	_ = cmd.Flags().Set("repo", "https://github.com/test/repo.git")

	err := ExecuteCommand(t, cmd, []string{"/some/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both")
}

// TestValidateCommand_WithCheckSystemFlag tests system requirements check.
func TestValidateCommand_WithCheckSystemFlag(t *testing.T) {
	tempDir := t.TempDir()

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{
			SystemRequirementsFunc: func() error {
				return assert.AnError
			},
		}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	_ = cmd.Flags().Set("check-system", "true")

	err := ExecuteCommand(t, cmd, []string{tempDir})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system requirements not met")
}

// TestValidateCommand_WithValidationErrors tests validation with errors.
func TestValidateCommand_WithValidationErrors(t *testing.T) {
	tempDir := t.TempDir()

	composeContent := `
services:
  test:
    image: nginx:latest
    build:
      context: ""
`
	err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeContent), 0600)
	require.NoError(t, err)

	app := NewAppBuilder(t).
		WithValidator(&MockValidator{}).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{tempDir})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestIsValidGitRepo tests git repository validation function.
func TestIsValidGitRepo(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0750)
	require.NoError(t, err)

	valid := isValidGitRepo(tempDir)
	assert.True(t, valid)

	valid = isValidGitRepo("/nonexistent")
	assert.False(t, valid)
}

// TestIsComposeFile tests compose file detection.
func TestIsComposeFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		expected bool
	}{
		{
			name:     "valid compose file",
			filename: "docker-compose.yml",
			content:  "services:\n  test:\n    image: nginx",
			expected: true,
		},
		{
			name:     "yaml file with version",
			filename: "compose.yaml",
			content:  "version: '3.8'\nservices:\n  test:\n    image: nginx",
			expected: true,
		},
		{
			name:     "non-yaml file",
			filename: "file.txt",
			content:  "not yaml",
			expected: false,
		},
		{
			name:     "yaml without compose markers",
			filename: "config.yml",
			content:  "key: value",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			result := isComposeFile(filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
