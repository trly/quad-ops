package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/trly/quad-ops/internal/config"
)

func TestInitCommand_Run(t *testing.T) {
	tests := []struct {
		name        string
		opts        InitOptions
		setup       func(t *testing.T, tempDir string)
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, tempDir string)
	}{
		{
			name: "creates config file successfully",
			opts: InitOptions{Force: false},
			setup: func(_ *testing.T, _ string) {
				// No setup needed
			},
			expectError: false,
			validate: func(t *testing.T, tempDir string) {
				configFile := filepath.Join(tempDir, ".config", "quad-ops", "config.yaml")
				assert.FileExists(t, configFile)

				data, err := os.ReadFile(configFile) // #nosec G304
				require.NoError(t, err)

				var cfg config.Settings
				err = yaml.Unmarshal(data, &cfg)
				require.NoError(t, err)

				require.Len(t, cfg.Repositories, 1)
				repo := cfg.Repositories[0]
				assert.Equal(t, "quad-ops-examples", repo.Name)
				assert.Equal(t, "https://github.com/trly/quad-ops.git", repo.URL)
				assert.Equal(t, "main", repo.Reference)
				assert.Equal(t, "examples", repo.ComposeDir)
			},
		},
		{
			name: "fails when config file exists and force is false",
			opts: InitOptions{Force: false},
			setup: func(_ *testing.T, tempDir string) {
				configDir := filepath.Join(tempDir, ".config", "quad-ops")
				require.NoError(t, os.MkdirAll(configDir, 0700))
				configFile := filepath.Join(configDir, "config.yaml")
				require.NoError(t, os.WriteFile(configFile, []byte("existing"), 0600))
			},
			expectError: true,
			errorMsg:    "configuration file already exists",
			validate: func(t *testing.T, tempDir string) {
				configFile := filepath.Join(tempDir, ".config", "quad-ops", "config.yaml")
				data, err := os.ReadFile(configFile) // #nosec G304
				require.NoError(t, err)
				assert.Equal(t, "existing", string(data))
			},
		},
		{
			name: "overwrites config file when force is true",
			opts: InitOptions{Force: true},
			setup: func(_ *testing.T, tempDir string) {
				configDir := filepath.Join(tempDir, ".config", "quad-ops")
				require.NoError(t, os.MkdirAll(configDir, 0700))
				configFile := filepath.Join(configDir, "config.yaml")
				require.NoError(t, os.WriteFile(configFile, []byte("existing"), 0600))
			},
			expectError: false,
			validate: func(t *testing.T, tempDir string) {
				configFile := filepath.Join(tempDir, ".config", "quad-ops", "config.yaml")
				assert.FileExists(t, configFile)

				data, err := os.ReadFile(configFile) // #nosec G304
				require.NoError(t, err)

				assert.NotEqual(t, "existing", string(data))
				assert.Contains(t, string(data), "quad-ops-examples")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory as home
			tempDir := t.TempDir()

			// Setup
			tt.setup(t, tempDir)

			// Create app (minimal)
			app := &App{
				Config: &config.Settings{UserMode: true},
			}

			// Create command
			cmd := NewInitCommand()

			// Build deps with mocked home dir
			deps := InitDeps{
				CommonDeps: CommonDeps{},
				UserHomeDir: func() (string, error) {
					return tempDir, nil
				},
				MkdirAll:  os.MkdirAll,
				WriteFile: os.WriteFile,
			}

			// Run
			err := cmd.Run(app, tt.opts, deps)

			// Assert
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, tempDir)
			}
		})
	}
}

func TestInitCommand_GetCobraCommand(t *testing.T) {
	cmd := NewInitCommand()
	cobraCmd := cmd.GetCobraCommand()

	assert.Equal(t, "init", cobraCmd.Use)
	assert.Equal(t, "Initialize a default configuration file", cobraCmd.Short)
	assert.Contains(t, cobraCmd.Long, "example repositories")

	// Check flags
	forceFlag := cobraCmd.Flag("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
	assert.Equal(t, "Overwrite existing configuration file", forceFlag.Usage)
}
