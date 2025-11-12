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

// TestConfigInit_DefaultBehavior tests default config initialization.
func TestConfigInit_DefaultBehavior(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "quad-ops")
	configFile := filepath.Join(configDir, "config.yaml")

	app := &App{
		Config: &config.Settings{UserMode: true},
	}

	cmd := NewInitCommand()
	deps := InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: func() (string, error) { return tempDir, nil },
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}

	err := cmd.Run(app, InitOptions{Force: false}, deps)
	require.NoError(t, err)

	// Verify config file created
	assert.FileExists(t, configFile)

	// Verify config content
	data, err := os.ReadFile(configFile) // #nosec G304
	require.NoError(t, err)

	var cfg config.Settings
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	require.Len(t, cfg.Repositories, 1)
	assert.Equal(t, "quad-ops-examples", cfg.Repositories[0].Name)
}

// TestConfigInit_ExistingConfigNoForce tests behavior when config exists without force.
func TestConfigInit_ExistingConfigNoForce(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "quad-ops")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create existing config
	require.NoError(t, os.MkdirAll(configDir, 0700))
	existingContent := []byte("existing: config")
	require.NoError(t, os.WriteFile(configFile, existingContent, 0600))

	app := &App{
		Config: &config.Settings{UserMode: true},
	}

	cmd := NewInitCommand()
	deps := InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: func() (string, error) { return tempDir, nil },
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}

	err := cmd.Run(app, InitOptions{Force: false}, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Verify original config unchanged
	data, err := os.ReadFile(configFile) // #nosec G304
	require.NoError(t, err)
	assert.Equal(t, existingContent, data)
}

// TestConfigInit_ExistingConfigWithForce tests force overwrite.
func TestConfigInit_ExistingConfigWithForce(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "quad-ops")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create existing config
	require.NoError(t, os.MkdirAll(configDir, 0700))
	existingContent := []byte("existing: config")
	require.NoError(t, os.WriteFile(configFile, existingContent, 0600))

	app := &App{
		Config: &config.Settings{UserMode: true},
	}

	cmd := NewInitCommand()
	deps := InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: func() (string, error) { return tempDir, nil },
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}

	err := cmd.Run(app, InitOptions{Force: true}, deps)
	require.NoError(t, err)

	// Verify config was overwritten
	data, err := os.ReadFile(configFile) // #nosec G304
	require.NoError(t, err)
	assert.NotEqual(t, existingContent, data)
	assert.Contains(t, string(data), "quad-ops-examples")
}

// TestConfigShow_DisplaysConfiguration tests config show command.
func TestConfigShow_DisplaysConfiguration(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose:       true,
			UserMode:      true,
			QuadletDir:    "/home/user/.config/containers/systemd",
			RepositoryDir: "/home/user/.local/share/quad-ops/repos",
			Repositories: []config.Repository{
				{Name: "test-app", URL: "https://example.com/test.git"},
			},
		}).
		Build(t)

	cmd := NewConfigShowCommand()
	cobraCmd := cmd.GetCobraCommand()
	SetupCommandContext(cobraCmd, app)

	output, err := ExecuteCommandWithCapture(t, cobraCmd, []string{})
	require.NoError(t, err)

	assert.Contains(t, output, "verbose: true")
	assert.Contains(t, output, "test-app")
}

// TestConfigShow_YAMLOutput tests YAML format output (default).
func TestConfigShow_YAMLOutput(t *testing.T) {
	app := NewAppBuilder(t).
		WithConfig(&config.Settings{
			Verbose:  false,
			UserMode: true,
			Repositories: []config.Repository{
				{Name: "app1", URL: "https://example.com/app1.git"},
			},
		}).
		Build(t)

	cmd := NewConfigShowCommand()
	cobraCmd := cmd.GetCobraCommand()
	SetupCommandContext(cobraCmd, app)

	output, err := ExecuteCommandWithCapture(t, cobraCmd, []string{})
	require.NoError(t, err)

	assert.Contains(t, output, "verbose:")
	assert.Contains(t, output, "app1")
}

// TestConfig_WorkflowInitShowUpdate tests full config workflow.
func TestConfig_WorkflowInitShowUpdate(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "quad-ops")
	configFile := filepath.Join(configDir, "config.yaml")

	// Step 1: Initialize config
	app := &App{
		Config: &config.Settings{UserMode: true},
	}

	initCmd := NewInitCommand()
	initDeps := InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: func() (string, error) { return tempDir, nil },
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}

	err := initCmd.Run(app, InitOptions{Force: false}, initDeps)
	require.NoError(t, err)
	assert.FileExists(t, configFile)

	// Step 2: Load and show config
	data, err := os.ReadFile(configFile) // #nosec G304
	require.NoError(t, err)

	var cfg config.Settings
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	app = NewAppBuilder(t).WithConfig(&cfg).Build(t)

	showCmd := NewConfigShowCommand()
	showCmdCobra := showCmd.GetCobraCommand()
	SetupCommandContext(showCmdCobra, app)

	output, err := ExecuteCommandWithCapture(t, showCmdCobra, []string{})
	require.NoError(t, err)
	assert.Contains(t, output, "quad-ops-examples")

	// Step 3: Modify config file
	cfg.Verbose = true
	cfg.Repositories = append(cfg.Repositories, config.Repository{
		Name: "my-app",
		URL:  "https://example.com/my-app.git",
	})

	updatedData, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	err = os.WriteFile(configFile, updatedData, 0600)
	require.NoError(t, err)

	// Step 4: Verify updated config shows correctly
	data, err = os.ReadFile(configFile) // #nosec G304
	require.NoError(t, err)

	var updatedCfg config.Settings
	err = yaml.Unmarshal(data, &updatedCfg)
	require.NoError(t, err)

	assert.True(t, updatedCfg.Verbose)
	assert.Len(t, updatedCfg.Repositories, 2)
	assert.Equal(t, "my-app", updatedCfg.Repositories[1].Name)
}

// TestConfig_DirectoryCreation tests config init creates necessary directories.
func TestConfig_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()

	app := &App{
		Config: &config.Settings{UserMode: true},
	}

	cmd := NewInitCommand()
	deps := InitDeps{
		CommonDeps:  CommonDeps{},
		UserHomeDir: func() (string, error) { return tempDir, nil },
		MkdirAll:    os.MkdirAll,
		WriteFile:   os.WriteFile,
	}

	err := cmd.Run(app, InitOptions{Force: false}, deps)
	require.NoError(t, err)

	// Verify directory structure created
	configDir := filepath.Join(tempDir, ".config", "quad-ops")
	assert.DirExists(t, configDir)

	configFile := filepath.Join(configDir, "config.yaml")
	assert.FileExists(t, configFile)

	// Verify permissions
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestConfig_InitHelp tests config init help output.
func TestConfig_InitHelp(t *testing.T) {
	cmd := NewInitCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "configuration file")
	assert.Contains(t, output, "--force")
	assert.Contains(t, output, "example repositories")
}

// TestConfig_ShowHelp tests config show help output.
func TestConfig_ShowHelp(t *testing.T) {
	cmd := NewConfigShowCommand().GetCobraCommand()
	output, err := ExecuteCommandWithCapture(t, cmd, []string{"--help"})

	require.NoError(t, err)
	assert.Contains(t, output, "configuration")
}
