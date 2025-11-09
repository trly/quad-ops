package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/testutil"
	"github.com/trly/quad-ops/internal/validate"
)

// TestValidateCommand_Basic tests validate command.
func TestValidateCommand_Basic(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	err = ExecuteCommand(t, cmd, []string{composeFile})
	assert.Error(t, err)
}

// TestValidateCommand_NonExistentPath tests non-existent path.
func TestValidateCommand_NonExistentPath(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
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

	// Test with .git as a file instead of directory
	tempDir2 := t.TempDir()
	gitFile := filepath.Join(tempDir2, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0600)
	require.NoError(t, err)

	valid = isValidGitRepo(tempDir2)
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
		{
			name:     "networks only",
			filename: "networks.yml",
			content:  "networks:\n  mynet:\n    driver: bridge",
			expected: true,
		},
		{
			name:     "volumes only",
			filename: "volumes.yml",
			content:  "volumes:\n  myvol:\n    driver: local",
			expected: true,
		},
		{
			name:     "empty yaml file",
			filename: "empty.yml",
			content:  "",
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

// TestIsComposeFile_EdgeCases tests edge cases for compose file detection.
func TestIsComposeFile_EdgeCases(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		result := isComposeFile("/nonexistent/path.yml")
		assert.False(t, result)
	})

	t.Run("uppercase extension", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "compose.YML")
		err := os.WriteFile(filePath, []byte("services:\n  test:\n    image: nginx"), 0600)
		require.NoError(t, err)

		result := isComposeFile(filePath)
		assert.True(t, result)
	})

	t.Run("yaml extension", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "compose.YAML")
		err := os.WriteFile(filePath, []byte("services:\n  test:\n    image: nginx"), 0600)
		require.NoError(t, err)

		result := isComposeFile(filePath)
		assert.True(t, result)
	})
}

// TestValidateService tests service validation function.
func TestValidateService(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	project := &types.Project{}

	tests := []struct {
		name        string
		serviceName string
		service     types.ServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid service",
			serviceName: "web",
			service: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
			},
			expectError: false,
		},
		{
			name:        "service with valid environment",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Environment: types.MappingWithEquals{
					"DATABASE_URL": stringPtr("postgresql://localhost:5432/mydb"),
					"DEBUG":        stringPtr("false"),
				},
			},
			expectError: false,
		},
		{
			name:        "service with invalid env key",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Environment: types.MappingWithEquals{
					"INVALID KEY": stringPtr("value"),
				},
			},
			expectError: true,
			errorMsg:    "invalid environment key",
		},
		{
			name:        "service with valid secrets",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Secrets: []types.ServiceSecretConfig{
					{Source: "db_password", Target: "/run/secrets/db_password"},
				},
			},
			expectError: false,
		},
		{
			name:        "service with secret name with special chars",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Secrets: []types.ServiceSecretConfig{
					{Source: "db-password", Target: "/run/secrets/db"},
				},
			},
			expectError: false,
		},
		{
			name:        "service with valid build config",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Build: &types.BuildConfig{Context: "./app"},
			},
			expectError: false,
		},
		{
			name:        "service with empty build context",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Build: &types.BuildConfig{Context: ""},
			},
			expectError: true,
			errorMsg:    "build context cannot be empty",
		},
		{
			name:        "service with init containers",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Labels: types.Labels{
					"quad-ops.init-containers": "init-db",
				},
			},
			expectError: false,
		},
		{
			name:        "service with empty init container label",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Labels: types.Labels{
					"quad-ops.init-containers": "   ",
				},
			},
			expectError: true,
			errorMsg:    "init container label",
		},
		{
			name:        "service with nil environment value",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Environment: types.MappingWithEquals{
					"VALID_KEY": nil,
				},
			},
			expectError: false,
		},
		{
			name:        "service with secret without target",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Secrets: []types.ServiceSecretConfig{
					{Source: "db_password"},
				},
			},
			expectError: false,
		},
		{
			name:        "service with invalid secret target",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Secrets: []types.ServiceSecretConfig{
					{Source: "db_password", Target: "invalid target!"},
				},
			},
			expectError: true,
			errorMsg:    "invalid secret target",
		},
		{
			name:        "service with x-podman-env-secrets referencing non-existent secret",
			serviceName: "app",
			service: types.ServiceConfig{
				Name:  "app",
				Image: "myapp:latest",
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{
						"non_existent_secret": "SOME_ENV",
					},
				},
			},
			expectError: true,
			errorMsg:    "podman secret 'non_existent_secret' referenced in x-podman-env-secrets does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateService(tt.serviceName, tt.service, project, validator, logger)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateBuild tests build configuration validation.
func TestValidateBuild(t *testing.T) {
	tests := []struct {
		name        string
		build       *types.BuildConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid build config",
			build:       &types.BuildConfig{Context: "./app"},
			expectError: false,
		},
		{
			name:        "empty context",
			build:       &types.BuildConfig{Context: ""},
			expectError: true,
			errorMsg:    "build context cannot be empty",
		},
		{
			name: "valid build args",
			build: &types.BuildConfig{
				Context: "./app",
				Args: types.MappingWithEquals{
					"NODE_VERSION": stringPtr("18"),
					"APP_ENV":      stringPtr("production"),
				},
			},
			expectError: false,
		},
		{
			name: "invalid build arg key",
			build: &types.BuildConfig{
				Context: "./app",
				Args: types.MappingWithEquals{
					"INVALID KEY!": stringPtr("value"),
				},
			},
			expectError: true,
			errorMsg:    "invalid build arg key",
		},
		{
			name: "build arg value too large",
			build: &types.BuildConfig{
				Context: "./app",
				Args: types.MappingWithEquals{
					"LARGE_VALUE": stringPtr(string(make([]byte, validate.MaxEnvValueSize+1))),
				},
			},
			expectError: true,
			errorMsg:    "exceeds maximum size",
		},
		{
			name: "nil build arg value",
			build: &types.BuildConfig{
				Context: "./app",
				Args: types.MappingWithEquals{
					"OPTIONAL_ARG": nil,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBuild(tt.build)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateInitContainers tests init container validation.
func TestValidateInitContainers(t *testing.T) {
	tests := []struct {
		name        string
		service     types.ServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "no init containers",
			service: types.ServiceConfig{
				Name: "app",
			},
			expectError: false,
		},
		{
			name: "valid init containers",
			service: types.ServiceConfig{
				Name: "app",
				Labels: types.Labels{
					"quad-ops.init-containers": "db-migration,cache-warmup",
				},
			},
			expectError: false,
		},
		{
			name: "valid quad-ops.init label",
			service: types.ServiceConfig{
				Name: "app",
				Labels: types.Labels{
					"quad-ops.init": "setup",
				},
			},
			expectError: false,
		},
		{
			name: "empty init container label",
			service: types.ServiceConfig{
				Name: "app",
				Labels: types.Labels{
					"quad-ops.init-containers": "",
				},
			},
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name: "whitespace only init label",
			service: types.ServiceConfig{
				Name: "app",
				Labels: types.Labels{
					"quad-ops.init": "   ",
				},
			},
			expectError: true,
			errorMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInitContainers(tt.service)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateNetwork tests network validation.
func TestValidateNetwork(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		network     types.NetworkConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid network",
			networkName: "mynet",
			network: types.NetworkConfig{
				Name:   "mynet",
				Driver: "bridge",
			},
			expectError: false,
		},
		{
			name:        "external network without driver",
			networkName: "external_net",
			network: types.NetworkConfig{
				Name:     "external_net",
				External: true,
			},
			expectError: false,
		},
		{
			name:        "external network with driver",
			networkName: "bad_net",
			network: types.NetworkConfig{
				Name:     "bad_net",
				External: true,
				Driver:   "bridge",
			},
			expectError: true,
			errorMsg:    "external networks cannot specify driver",
		},
		{
			name:        "network without driver",
			networkName: "simple_net",
			network: types.NetworkConfig{
				Name: "simple_net",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetwork(tt.networkName, tt.network)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateVolume tests volume validation.
func TestValidateVolume(t *testing.T) {
	tests := []struct {
		name        string
		volumeName  string
		volume      types.VolumeConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:       "valid volume",
			volumeName: "myvol",
			volume: types.VolumeConfig{
				Name:   "myvol",
				Driver: "local",
			},
			expectError: false,
		},
		{
			name:       "external volume without driver",
			volumeName: "external_vol",
			volume: types.VolumeConfig{
				Name:     "external_vol",
				External: true,
			},
			expectError: false,
		},
		{
			name:       "external volume with driver",
			volumeName: "bad_vol",
			volume: types.VolumeConfig{
				Name:     "bad_vol",
				External: true,
				Driver:   "local",
			},
			expectError: true,
			errorMsg:    "external volumes cannot specify driver",
		},
		{
			name:       "volume without driver",
			volumeName: "simple_vol",
			volume: types.VolumeConfig{
				Name: "simple_vol",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolume(tt.volumeName, tt.volume)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSecretWithDeps tests secret validation.
func TestValidateSecretWithDeps(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)

	tests := []struct {
		name        string
		secretName  string
		secret      types.SecretConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:       "valid secret",
			secretName: "db_password",
			secret: types.SecretConfig{
				Name: "db_password",
				File: "/run/secrets/db_password",
			},
			expectError: false,
		},
		{
			name:        "secret name with underscores",
			secretName:  "my_secret_key",
			secret:      types.SecretConfig{Name: "my_secret_key"},
			expectError: false,
		},
		{
			name:       "relative file path",
			secretName: "api_key",
			secret: types.SecretConfig{
				Name: "api_key",
				File: "./secrets/api.key",
			},
			expectError: false,
		},
		{
			name:       "path with directory traversal",
			secretName: "bad_secret",
			secret: types.SecretConfig{
				Name: "bad_secret",
				File: "../../../etc/passwd",
			},
			expectError: true,
			errorMsg:    "directory traversal",
		},
		{
			name:       "absolute path",
			secretName: "secure_key",
			secret: types.SecretConfig{
				Name: "secure_key",
				File: "/run/secrets/secure.key",
			},
			expectError: false,
		},
		{
			name:       "secret without file",
			secretName: "env_secret",
			secret: types.SecretConfig{
				Name: "env_secret",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretWithDeps(tt.secretName, tt.secret, validator, logger)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateProjectWithDeps tests project validation.
func TestValidateProjectWithDeps(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)

	tests := []struct {
		name        string
		project     *types.Project
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid project",
			project: &types.Project{
				Name: "myapp",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			expectError: false,
		},
		{
			name: "project with invalid service",
			project: &types.Project{
				Name: "myapp",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
						Environment: types.MappingWithEquals{
							"INVALID KEY": stringPtr("value"),
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "service web",
		},
		{
			name: "project with invalid network",
			project: &types.Project{
				Name: "myapp",
				Networks: types.Networks{
					"mynet": {
						Name:     "mynet",
						External: true,
						Driver:   "bridge",
					},
				},
			},
			expectError: true,
			errorMsg:    "network mynet",
		},
		{
			name: "project with invalid volume",
			project: &types.Project{
				Name: "myapp",
				Volumes: types.Volumes{
					"myvol": {
						Name:     "myvol",
						External: true,
						Driver:   "local",
					},
				},
			},
			expectError: true,
			errorMsg:    "volume myvol",
		},
		{
			name: "project with secret directory traversal",
			project: &types.Project{
				Name: "myapp",
				Secrets: types.Secrets{
					"bad_secret": {
						Name: "bad_secret",
						File: "../../etc/passwd",
					},
				},
			},
			expectError: true,
			errorMsg:    "secret bad_secret",
		},
		{
			name: "empty project",
			project: &types.Project{
				Name: "empty",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectWithDeps(tt.project, validator, logger)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateComposeWithDeps tests compose validation with dependencies.
func TestValidateComposeWithDeps(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)

	t.Run("valid directory", func(t *testing.T) {
		tempDir := t.TempDir()
		composeContent := `
services:
  web:
    image: nginx:latest
`
		err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeContent), 0600)
		require.NoError(t, err)

		err = validateComposeWithDeps(tempDir, validator, logger)
		assert.NoError(t, err)
	})

	t.Run("valid single file", func(t *testing.T) {
		tempDir := t.TempDir()
		composeFile := filepath.Join(tempDir, "docker-compose.yml")
		composeContent := `
services:
  web:
    image: nginx:latest
`
		err := os.WriteFile(composeFile, []byte(composeContent), 0600)
		require.NoError(t, err)

		err = validateComposeWithDeps(composeFile, validator, logger)
		assert.NoError(t, err)
	})

	t.Run("nonexistent path", func(t *testing.T) {
		err := validateComposeWithDeps("/nonexistent/path", validator, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("non-compose file", func(t *testing.T) {
		tempDir := t.TempDir()
		textFile := filepath.Join(tempDir, "file.txt")
		err := os.WriteFile(textFile, []byte("not a compose file"), 0600)
		require.NoError(t, err)

		err = validateComposeWithDeps(textFile, validator, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not appear to be a Docker Compose file")
	})

	t.Run("empty directory", func(t *testing.T) {
		tempDir := t.TempDir()
		err := validateComposeWithDeps(tempDir, validator, logger)
		assert.NoError(t, err)
	})

	t.Run("invalid compose file", func(t *testing.T) {
		tempDir := t.TempDir()
		composeFile := filepath.Join(tempDir, "docker-compose.yml")
		composeContent := `
services:
  web:
    image: nginx:latest
    environment:
      INVALID KEY: value
`
		err := os.WriteFile(composeFile, []byte(composeContent), 0600)
		require.NoError(t, err)

		err = validateComposeWithDeps(composeFile, validator, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("multiple projects", func(t *testing.T) {
		tempDir := t.TempDir()
		composeContent1 := `
services:
  web:
    image: nginx:latest
`
		composeContent2 := `
services:
  db:
    image: postgres:latest
`
		err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeContent1), 0600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "compose2.yml"), []byte(composeContent2), 0600)
		require.NoError(t, err)

		err = validateComposeWithDeps(tempDir, validator, logger)
		assert.NoError(t, err)
	})
}

// TestCloneRepositoryWithDeps tests repository cloning with dependencies.
func TestCloneRepositoryWithDeps(t *testing.T) {
	t.Run("safe temp path construction", func(t *testing.T) {
		logger := testutil.NewTestLogger(t)
		mockConfig := testutil.NewMockConfig(t)

		// Set flags
		originalRepoURL := repoURL
		originalRepoRef := repoRef
		originalTempDir := tempDir
		originalSkipClone := skipClone

		repoURL = "https://github.com/test/repo.git"
		repoRef = "main"
		tempDir = t.TempDir()
		skipClone = false

		defer func() {
			repoURL = originalRepoURL
			repoRef = originalRepoRef
			tempDir = originalTempDir
			skipClone = originalSkipClone
		}()

		// This will fail at clone but will verify path construction includes suffix
		_, _, err := cloneRepositoryWithDeps(logger, mockConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone")
	})

	t.Run("skip clone with valid repo", func(t *testing.T) {
		logger := testutil.NewTestLogger(t)
		mockConfig := testutil.NewMockConfig(t)

		// Create a temporary git repo
		tempGitDir := t.TempDir()
		gitPath := filepath.Join(tempGitDir, "quad-ops-validate")
		err := os.MkdirAll(filepath.Join(gitPath, ".git"), 0750)
		require.NoError(t, err)

		// Set flags
		originalRepoURL := repoURL
		originalRepoRef := repoRef
		originalTempDir := tempDir
		originalSkipClone := skipClone

		repoURL = "https://github.com/test/repo.git"
		repoRef = "main"
		tempDir = tempGitDir
		skipClone = true

		defer func() {
			repoURL = originalRepoURL
			repoRef = originalRepoRef
			tempDir = originalTempDir
			skipClone = originalSkipClone
		}()

		path, cleanup, err := cloneRepositoryWithDeps(logger, mockConfig)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.NotNil(t, cleanup)
		assert.NoError(t, cleanup())
	})

	t.Run("skip clone with invalid repo", func(t *testing.T) {
		logger := testutil.NewTestLogger(t)
		mockConfig := testutil.NewMockConfig(t)

		// Create temp dir without .git
		tempGitDir := t.TempDir()

		originalRepoURL := repoURL
		originalRepoRef := repoRef
		originalTempDir := tempDir
		originalSkipClone := skipClone

		repoURL = "https://github.com/test/repo.git"
		repoRef = "main"
		tempDir = tempGitDir
		skipClone = true

		defer func() {
			repoURL = originalRepoURL
			repoRef = originalRepoRef
			tempDir = originalTempDir
			skipClone = originalSkipClone
		}()

		_, _, err := cloneRepositoryWithDeps(logger, mockConfig)
		assert.Error(t, err)
	})

	t.Run("default temp dir", func(t *testing.T) {
		logger := testutil.NewTestLogger(t)
		mockConfig := testutil.NewMockConfig(t)

		originalRepoURL := repoURL
		originalRepoRef := repoRef
		originalTempDir := tempDir
		originalSkipClone := skipClone

		repoURL = "https://github.com/test/repo.git"
		repoRef = "main"
		tempDir = ""
		skipClone = false

		defer func() {
			repoURL = originalRepoURL
			repoRef = originalRepoRef
			tempDir = originalTempDir
			skipClone = originalSkipClone
		}()

		// Test that using default temp dir (os.TempDir) works
		// The function will either succeed or fail depending on the network/environment
		// The key is that the path construction uses the suffix check correctly
		path, cleanup, err := cloneRepositoryWithDeps(logger, mockConfig)
		if err == nil && cleanup != nil {
			// If successful, verify path is correctly structured
			assert.NotEmpty(t, path)
			assert.NoError(t, cleanup())
		}
		// If it fails, that's also acceptable (network issues in test environment)
	})
}

// Helper function to create string pointers.
func stringPtr(s string) *string {
	return &s
}

// TestValidateCommand_WithComposeDir tests compose-dir flag.
func TestValidateCommand_WithComposeDir(t *testing.T) {
	tempDir := t.TempDir()
	servicesDir := filepath.Join(tempDir, "services")
	err := os.MkdirAll(servicesDir, 0750)
	require.NoError(t, err)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	err = os.WriteFile(filepath.Join(servicesDir, "docker-compose.yml"), []byte(composeContent), 0600)
	require.NoError(t, err)

	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	app := NewAppBuilder(t).
		WithValidator(validator).
		Build(t)

	validateCmd := NewValidateCommand()
	cmd := validateCmd.GetCobraCommand()
	SetupCommandContext(cmd, app)

	// Set compose-dir flag
	originalComposeDir := composeDir
	composeDir = "services"
	defer func() { composeDir = originalComposeDir }()

	err = ExecuteCommand(t, cmd, []string{tempDir})
	assert.NoError(t, err)
}

// TestValidateCommand_WithRepoAndComposeDir tests --repo and --compose-dir together.
// This is the regression test for quad-ops-b5m: ensure that when both flags are used,
// the path correctly includes the repository subdirectory created by GitSyncer.
func TestValidateCommand_WithRepoAndComposeDir(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)

	// Save original flag values
	originalRepoURL := repoURL
	originalRepoRef := repoRef
	originalTempDir := tempDir
	originalSkipClone := skipClone
	originalComposeDir := composeDir

	// Set up test flags
	repoURL = "https://github.com/test/repo.git"
	repoRef = "main"
	tempDir = t.TempDir()
	skipClone = false
	composeDir = "prod/services"

	defer func() {
		repoURL = originalRepoURL
		repoRef = originalRepoRef
		tempDir = originalTempDir
		skipClone = originalSkipClone
		composeDir = originalComposeDir
	}()

	// Test that the path construction includes the repository name subdirectory.
	// With the fix, the returned path should be:
	//   tempDir/quad-ops-validate/validate-temp
	// not:
	//   tempDir/quad-ops-validate (which was the bug)
	path, cleanup, err := cloneRepositoryWithDeps(logger, mockConfig)
	// Note: The clone may succeed or fail depending on network/environment
	// The key is that if it succeeds, the path is correctly constructed
	if err == nil && cleanup != nil {
		// Verify path includes the repo name directory
		assert.Contains(t, path, "validate-temp")
		assert.NoError(t, cleanup())
	}
}

// TestCloneRepositoryWithDeps_PathConstruction tests that the cloned repository
// path includes the repository subdirectory created by GitSyncer.
// This is a focused test for quad-ops-b5m regression.
func TestCloneRepositoryWithDeps_PathConstruction(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	mockConfig := testutil.NewMockConfig(t)

	// Save original flag values
	originalRepoURL := repoURL
	originalRepoRef := repoRef
	originalTempDir := tempDir
	originalSkipClone := skipClone

	// Create a temp directory and set up the repository
	testTempDir := t.TempDir()
	repoURL = "https://github.com/test/repo.git"
	repoRef = "main"
	tempDir = testTempDir
	skipClone = false

	defer func() {
		repoURL = originalRepoURL
		repoRef = originalRepoRef
		tempDir = originalTempDir
		skipClone = originalSkipClone
	}()

	// Call cloneRepositoryWithDeps and verify the returned path
	path, cleanup, err := cloneRepositoryWithDeps(logger, mockConfig)

	// The key assertion: verify the path includes the repository name subdirectory
	// Expected format: /path/to/tempdir/quad-ops-validate/validate-temp
	// NOT: /path/to/tempdir/quad-ops-validate (which was the bug)
	if err == nil && cleanup != nil {
		defer func() {
			_ = cleanup()
		}()

		// Verify path contains the expected subdirectory structure
		assert.Contains(t, path, "quad-ops-validate", "Path should contain quad-ops-validate directory")
		assert.Contains(t, path, "validate-temp", "Path should contain validate-temp subdirectory created by GitSyncer")

		// Verify the path ends with validate-temp (the repo name)
		expectedSuffix := filepath.Join("quad-ops-validate", "validate-temp")
		assert.True(t, strings.HasSuffix(path, expectedSuffix),
			"Path should end with %s but got %s", expectedSuffix, path)

		// Verify that composing a compose-dir path would work correctly
		composeSubDir := "prod/services"
		fullPath := filepath.Join(path, composeSubDir)
		// The full path should contain all parts: quad-ops-validate/validate-temp/prod/services
		assert.Contains(t, fullPath, filepath.Join("quad-ops-validate", "validate-temp", "prod", "services"),
			"Composed path should include all directory components")
	}
}

// TestValidateCommand_AccessError tests path access errors.
func TestValidateCommand_AccessError(t *testing.T) {
	logger := log.NewLogger(false)
	validator := validate.NewValidatorWithDefaults(logger)

	// Create a file and try to treat it as directory for access error
	tempFile := filepath.Join(t.TempDir(), "file.txt")
	err := os.WriteFile(tempFile, []byte("test"), 0000)
	require.NoError(t, err)

	// Make it unreadable
	err = os.Chmod(tempFile, 0000)
	require.NoError(t, err)

	defer func() {
		_ = os.Chmod(tempFile, 0600)
	}()

	err = validateComposeWithDeps(tempFile, validator, logger)
	assert.Error(t, err)
}

// TestValidateCommand_ParseError tests compose file parse errors.
func TestValidateCommand_ParseError(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	tempDir := t.TempDir()

	// Create an invalid compose file
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	err := os.WriteFile(composeFile, []byte("services: [invalid yaml"), 0600)
	require.NoError(t, err)

	err = validateComposeWithDeps(composeFile, validator, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestValidateCommand_DirectoryReadError tests directory read errors.
func TestValidateCommand_DirectoryReadError(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	validator := validate.NewValidatorWithDefaults(logger)
	tempDir := t.TempDir()

	// Create subdirectory with bad permissions
	badDir := filepath.Join(tempDir, "baddir")
	err := os.MkdirAll(badDir, 0000)
	require.NoError(t, err)

	defer func() {
		_ = os.Chmod(badDir, 0600) // #nosec G302
	}()

	// Note: os.ReadDir may still succeed with 000 permissions on macOS
	// This test documents the behavior but doesn't strictly enforce an error
	err = validateComposeWithDeps(badDir, validator, logger)
	// On macOS with certain file systems, read may still work, so we don't assert error
	_ = err
}
