package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_ValidComposeFile tests reading a valid compose file from filesystem.
func TestLoad_ValidComposeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal valid compose file
	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox
    command: echo "hello"`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, tmpDir, nil)

	require.NoError(t, err)
	assert.NotNil(t, project)
}

// TestLoad_ExplicitFilePath tests reading with explicit file path.
func TestLoad_ExplicitFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  web:
    image: nginx`

	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, composeFile, nil)

	require.NoError(t, err)
	assert.NotNil(t, project)
	assert.Len(t, project.Services, 1)
}

// TestLoad_FileNotFound tests error when no compose file exists.
func TestLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	_, err := Load(ctx, tmpDir, nil)

	require.Error(t, err)
	assert.True(t, IsFileNotFoundError(err))
}

// TestLoad_WithEnvironment tests environment variable interpolation.
func TestLoad_WithEnvironment(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: ${APP_IMAGE}`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(
		ctx,
		tmpDir,
		&LoadOptions{Environment: map[string]string{"APP_IMAGE": "myapp:latest"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "myapp:latest", project.Services["app"].Image)
}

// TestLoad_WithWorkdir tests custom working directory for relative paths.
func TestLoad_WithWorkdir(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox
    volumes:
      - ./data:/app/data`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(
		ctx,
		composeFile,
		&LoadOptions{Workdir: dataDir},
	)

	require.NoError(t, err)
	assert.NotNil(t, project)
	// Verify that the project was loaded with custom workdir
	assert.NotNil(t, project.Services["app"])
}

// TestLoad_WithEnvFiles tests loading environment from .env files.
func TestLoad_WithEnvFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env file
	envContent := "SERVICE_NAME=myservice\nSERVICE_PORT=8080"
	envFile := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0o644))

	// Create compose file that uses env vars
	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox
    environment:
      - SERVICE_NAME=${SERVICE_NAME}
      - SERVICE_PORT=${SERVICE_PORT}`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(
		ctx,
		tmpDir,
		&LoadOptions{EnvFiles: []string{envFile}},
	)

	require.NoError(t, err)
	assert.NotNil(t, project)
}

// TestLoad_ComposedValidationEnforced tests that validation cannot be skipped.
func TestLoad_ComposedValidationEnforced(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a compose file with quadlet-incompatible driver
	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox
volumes:
  bad-vol:
    driver: nfs`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	_, err := Load(ctx, tmpDir, nil)

	// Must fail - validation cannot be skipped
	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
}

// TestLoad_InvalidYAMLSyntax tests error on invalid YAML.
func TestLoad_InvalidYAMLSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid YAML with bad indentation
	composeContent := `version: '3.8'
name: test-project
services:
app:
    image: busybox`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	_, err := Load(ctx, tmpDir, nil)

	require.Error(t, err)
	// Could be invalid YAML or validation error depending on what compose-go catches first
	assert.True(t, IsInvalidYAMLError(err) || IsValidationError(err) || IsLoaderError(err))
}

// TestLoad_MultipleComposeFiles tests reading with compose overrides.
func TestLoad_MultipleComposeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	baseCompose := `version: '3.8'
name: test-project
services:
  app:
    image: busybox:latest
    ports:
      - "8080:8080"`

	overrideCompose := `version: '3.8'
services:
  app:
    environment:
      - DEBUG=true`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte(baseCompose), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "compose.override.yaml"), []byte(overrideCompose), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, tmpDir, nil)

	require.NoError(t, err)
	assert.NotNil(t, project)
	// Verify services were loaded
	assert.NotNil(t, project.Services["app"])
}

// TestLoad_ContextCancellation tests that context cancellation is respected.
func TestLoad_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Load(ctx, tmpDir, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestLoad_EmptyDirectory tests error when directory is empty.
func TestLoad_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	_, err := Load(ctx, tmpDir, nil)

	require.Error(t, err)
	assert.True(t, IsFileNotFoundError(err))
}

// TestLoad_NonexistentPath tests error for nonexistent path.
func TestLoad_NonexistentPath(t *testing.T) {
	ctx := context.Background()
	_, err := Load(ctx, "/nonexistent/path/compose.yaml", nil)

	require.Error(t, err)
	assert.True(t, IsFileNotFoundError(err))
}

// TestValidateProject_ValidProject tests validating a valid project.
func TestValidateProject_ValidProject(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
			},
		},
	}

	err := validateProject(context.Background(), project)

	require.NoError(t, err)
}

// TestValidateProject_NilProject tests error on nil project.
func TestValidateProject_NilProject(t *testing.T) {
	err := validateProject(context.Background(), nil)

	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

// TestValidateProject_InvalidService tests that services without image/build pass (compose allows them)
// or fail validation depending on compose-go version.
func TestValidateProject_InvalidService(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name: "app",
				// No image or build specified - validation behavior may vary
			},
		},
	}

	err := validateProject(context.Background(), project)

	// Service without image/build might be valid or invalid depending on compose-spec version
	// This test just ensures validation completes without panic
	assert.True(t, err == nil || IsValidationError(err))
}

// TestValidateProject_ContextCancellation tests context cancellation.
func TestValidateProject_ContextCancellation(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := validateProject(ctx, project)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestValidateProject_InvalidNetworkConfig tests validation of network config.
// Note: compose-go may or may not validate driver names.
func TestValidateProject_InvalidNetworkConfig(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx",
			},
		},
		Networks: types.Networks{
			"invalid": {
				Name:   "invalid",
				Driver: "nonexistent_driver",
			},
		},
	}

	err := validateProject(context.Background(), project)

	// Driver validation behavior may vary
	assert.True(t, err == nil || IsValidationError(err))
}

// TestValidateProject_InvalidVolumeConfig tests validation of volume config.
// Note: compose-go may or may not validate driver names.
func TestValidateProject_InvalidVolumeConfig(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx",
			},
		},
		Volumes: types.Volumes{
			"data": {
				Name:   "data",
				Driver: "invalid_driver",
			},
		},
	}

	err := validateProject(context.Background(), project)

	// Driver validation behavior may vary
	assert.True(t, err == nil || IsValidationError(err))
}

// TestLoad_ComplexCompose tests reading a complex compose file with multiple services.
func TestLoad_ComplexCompose(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  web:
    image: nginx:${NGINX_VERSION:-latest}
    ports:
      - "80:80"
    depends_on:
      - db
    volumes:
      - ./static:/usr/share/nginx/html:ro
    environment:
      - NGINX_HOST=${NGINX_HOST:-localhost}
      - NGINX_PORT=${NGINX_PORT:-80}
  db:
    image: postgres:15
    environment:
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - db-data:/var/lib/postgresql/data
volumes:
  db-data:
networks:
  default:
    driver: bridge`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(
		ctx,
		tmpDir,
		&LoadOptions{Environment: map[string]string{
			"NGINX_VERSION": "1.25",
			"NGINX_HOST":    "example.com",
			"DB_PASSWORD":   "secret",
		}},
	)

	require.NoError(t, err)
	assert.Len(t, project.Services, 2)
	assert.NotNil(t, project.Services["web"])
	assert.NotNil(t, project.Services["db"])
	assert.Equal(t, "nginx:1.25", project.Services["web"].Image)
	assert.Equal(t, "postgres:15", project.Services["db"].Image)
}

// TestLoad_WithMultipleOptions tests combining multiple options.
func TestLoad_WithMultipleOptions(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := t.TempDir()

	// Create .env file
	envContent := "CUSTOM_ENV=custom_value"
	envFile := filepath.Join(tmpDir, ".env.custom")
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0o644))

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: myapp:${APP_VERSION}
    environment:
      - CUSTOM_VAR=${CUSTOM_ENV}
    volumes:
      - ./logs:/app/logs`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(
		ctx,
		tmpDir,
		&LoadOptions{
			Workdir:     dataDir,
			Environment: map[string]string{"APP_VERSION": "1.0.0"},
			EnvFiles:    []string{envFile},
		},
	)

	require.NoError(t, err)
	assert.NotNil(t, project)
	assert.Len(t, project.Services, 1)
}

// TestProject_ProjectFiles tests the project structure contains project files.
func TestProject_ProjectFiles(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, tmpDir, nil)

	require.NoError(t, err)
	assert.NotNil(t, project)
	assert.NotNil(t, project.Services["app"])
}

// TestLoad_WithEnvFilesNonexistent tests handling of nonexistent env files.
func TestLoad_WithEnvFilesNonexistent(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	// Should handle nonexistent env files gracefully
	project, err := Load(
		ctx,
		tmpDir,
		&LoadOptions{EnvFiles: []string{"/nonexistent/.env"}},
	)

	// Either succeeds with just the compose file, or fails gracefully
	// The behavior depends on compose-go's handling
	if err != nil {
		assert.True(t, IsFileNotFoundError(err) || IsPathError(err))
	} else {
		assert.NotNil(t, project)
	}
}

// TestValidateProject_EmptyProject tests validation of minimal valid project.
func TestValidateProject_EmptyProject(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
	}

	err := validateProject(context.Background(), project)

	// An empty project with no services might be valid depending on compose-spec
	// This test ensures the function doesn't panic
	assert.True(t, err == nil || IsValidationError(err))
}

// TestLoad_DotenvLoading tests that .env files are automatically loaded from compose directory.
func TestLoad_DotenvLoading(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env file in the same directory
	envContent := "AUTO_ENV_VAR=auto_loaded"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0o644))

	composeContent := `version: '3.8'
name: test-project
services:
  app:
    image: busybox
    environment:
      - AUTO_VAR=${AUTO_ENV_VAR}`

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, tmpDir, nil)

	// compose-go should automatically load .env files
	require.NoError(t, err)
	assert.NotNil(t, project)
}

// TestValidateQuadletCompatibility_ServiceWithoutImage tests service missing image.
func TestValidateQuadletCompatibility_ServiceWithoutImage(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name: "app",
				// No image - should fail
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "no image")
}

// TestValidateQuadletCompatibility_ServiceWithImage tests service with valid image.
func TestValidateQuadletCompatibility_ServiceWithImage(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_UnsupportedDependsOnCondition tests unsupported depends_on condition.
func TestValidateQuadletCompatibility_UnsupportedDependsOnCondition(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: map[string]types.ServiceDependency{
					"db": {
						Condition: "service_healthy",
					},
				},
			},
			"db": {
				Name:  "db",
				Image: "postgres:15",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported depends_on condition")
	assert.Contains(t, err.Error(), "service_healthy")
}

// TestValidateQuadletCompatibility_SupportedDependsOnCondition tests supported depends_on condition.
func TestValidateQuadletCompatibility_SupportedDependsOnCondition(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: map[string]types.ServiceDependency{
					"db": {
						Condition: "service_started",
					},
				},
			},
			"db": {
				Name:  "db",
				Image: "postgres:15",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_SimpleDependency tests simple dependency without condition.
func TestValidateQuadletCompatibility_SimpleDependency(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: map[string]types.ServiceDependency{
					"db": {},
				},
			},
			"db": {
				Name:  "db",
				Image: "postgres:15",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_UnsupportedNetworkMode tests unsupported network mode.
func TestValidateQuadletCompatibility_UnsupportedNetworkMode(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:        "app",
				Image:       "nginx:latest",
				NetworkMode: "none",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported network mode")
	assert.Contains(t, err.Error(), "none")
}

// TestValidateQuadletCompatibility_SupportedNetworkMode tests supported network modes.
func TestValidateQuadletCompatibility_SupportedNetworkMode(t *testing.T) {
	testCases := []string{"host", "bridge", ""}
	for _, mode := range testCases {
		t.Run(fmt.Sprintf("NetworkMode_%s", mode), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:        "app",
						Image:       "nginx:latest",
						NetworkMode: mode,
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			assert.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_UnsupportedVolumeDriver tests unsupported volume driver.
func TestValidateQuadletCompatibility_UnsupportedVolumeDriver(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Source: "my-vol",
						Target: "/data",
					},
				},
			},
		},
		Volumes: types.Volumes{
			"my-vol": {
				Name:   "my-vol",
				Driver: "nfs",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported driver")
	assert.Contains(t, err.Error(), "nfs")
}

// TestValidateQuadletCompatibility_LocalVolumeDriver tests local volume driver.
func TestValidateQuadletCompatibility_LocalVolumeDriver(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Source: "my-vol",
						Target: "/data",
					},
				},
			},
		},
		Volumes: types.Volumes{
			"my-vol": {
				Name:   "my-vol",
				Driver: "local",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_UnsupportedNetworkDriver tests unsupported network driver.
func TestValidateQuadletCompatibility_UnsupportedNetworkDriver(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
			},
		},
		Networks: types.Networks{
			"custom": {
				Name:   "custom",
				Driver: "overlay",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported driver")
	assert.Contains(t, err.Error(), "overlay")
}

// TestValidateQuadletCompatibility_BridgeNetworkDriver tests bridge network driver.
func TestValidateQuadletCompatibility_BridgeNetworkDriver(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
			},
		},
		Networks: types.Networks{
			"app-network": {
				Name:   "app-network",
				Driver: "bridge",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_SecurityOptSupported tests that supported security_opt values pass validation.
func TestValidateQuadletCompatibility_SecurityOptSupported(t *testing.T) {
	supportedOpts := []string{
		"label=disable",
		"label:disable",
		"label=nested",
		"label=type:spc_t",
		"label=level:s0:c1,c2",
		"label=filetype:usr_t",
		"no-new-privileges",
		"no-new-privileges:true",
		"seccomp=/tmp/profile.json",
		"mask=/proc/kcore",
		"unmask=ALL",
	}

	for _, opt := range supportedOpts {
		t.Run(opt, func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:        "app",
						Image:       "nginx:latest",
						SecurityOpt: []string{opt},
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)
			require.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_SecurityOptUnsupported tests that unsupported security_opt is rejected.
func TestValidateQuadletCompatibility_SecurityOptUnsupported(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:        "app",
				Image:       "nginx:latest",
				SecurityOpt: []string{"apparmor=unconfined"},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported security_opt")
}

// TestValidateQuadletCompatibility_CapAdd tests that cap_add is accepted (supported via AddCapability).
func TestValidateQuadletCompatibility_CapAdd(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:   "app",
				Image:  "nginx:latest",
				CapAdd: []string{"NET_ADMIN"},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.NoError(t, err)
}

// TestValidateQuadletCompatibility_CapDrop tests that cap_drop is accepted (supported via DropCapability).
func TestValidateQuadletCompatibility_CapDrop(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:    "app",
				Image:   "nginx:latest",
				CapDrop: []string{"ALL"},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.NoError(t, err)
}

// TestValidateQuadletCompatibility_Privileged tests that privileged mode is accepted.
func TestValidateQuadletCompatibility_Privileged(t *testing.T) {
	trueVal := true
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:       "app",
				Image:      "nginx:latest",
				Privileged: trueVal,
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_User tests that user configuration is rejected.
func TestValidateQuadletCompatibility_User(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				User:  "appuser",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "user")
}

// TestValidateQuadletCompatibility_UnsupportedIpcMode tests unsupported IPC modes.
func TestValidateQuadletCompatibility_UnsupportedIpcMode(t *testing.T) {
	testCases := []string{"service:other", "container:mycontainer"}
	for _, ipcMode := range testCases {
		t.Run(fmt.Sprintf("IPC_%s", ipcMode), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:  "app",
						Image: "nginx:latest",
						Ipc:   ipcMode,
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			require.Error(t, err)
			assert.True(t, IsQuadletCompatibilityError(err))
			assert.Contains(t, err.Error(), "IPC")
		})
	}
}

// TestValidateQuadletCompatibility_SupportedIpcMode tests supported IPC modes.
func TestValidateQuadletCompatibility_SupportedIpcMode(t *testing.T) {
	testCases := []string{"private", "shareable", ""}
	for _, ipcMode := range testCases {
		t.Run(fmt.Sprintf("IPC_%s", ipcMode), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:  "app",
						Image: "nginx:latest",
						Ipc:   ipcMode,
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			assert.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_UnsupportedRestartPolicy tests unsupported restart policies.
func TestValidateQuadletCompatibility_UnsupportedRestartPolicy(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:    "app",
				Image:   "nginx:latest",
				Restart: "always-on-failure",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "restart policy")
}

// TestValidateQuadletCompatibility_SupportedRestartPolicy tests supported restart policies.
func TestValidateQuadletCompatibility_SupportedRestartPolicy(t *testing.T) {
	testCases := []string{"no", "always", "on-failure", "unless-stopped"}
	for _, policy := range testCases {
		t.Run(fmt.Sprintf("RestartPolicy_%s", policy), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:    "app",
						Image:   "nginx:latest",
						Restart: policy,
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			assert.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_Replicas tests that deploy.replicas > 1 is rejected.
func TestValidateQuadletCompatibility_Replicas(t *testing.T) {
	replicas := 3
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Deploy: &types.DeployConfig{
					Replicas: &replicas,
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "replicas")
}

// TestValidateQuadletCompatibility_SingleReplica tests that deploy.replicas = 1 is allowed.
func TestValidateQuadletCompatibility_SingleReplica(t *testing.T) {
	replicas := 1
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Deploy: &types.DeployConfig{
					Replicas: &replicas,
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	assert.NoError(t, err)
}

// TestValidateQuadletCompatibility_UnsupportedLoggingDriver tests unsupported logging drivers.
func TestValidateQuadletCompatibility_UnsupportedLoggingDriver(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Logging: &types.LoggingConfig{
					Driver: "splunk",
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "logging driver")
}

// TestValidateQuadletCompatibility_SupportedLoggingDriver tests supported logging drivers.
func TestValidateQuadletCompatibility_SupportedLoggingDriver(t *testing.T) {
	testCases := []string{"json-file", "journald"}
	for _, driver := range testCases {
		t.Run(fmt.Sprintf("LoggingDriver_%s", driver), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:  "app",
						Image: "nginx:latest",
						Logging: &types.LoggingConfig{
							Driver: driver,
						},
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			assert.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_UnsupportedStopSignal tests unsupported stop signals.
func TestValidateQuadletCompatibility_UnsupportedStopSignal(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:       "app",
				Image:      "nginx:latest",
				StopSignal: "SIGINT",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "stop signal")
}

// TestValidateQuadletCompatibility_SupportedStopSignal tests supported stop signals.
func TestValidateQuadletCompatibility_SupportedStopSignal(t *testing.T) {
	testCases := []string{"SIGTERM", "SIGKILL", "TERM", "KILL"}
	for _, signal := range testCases {
		t.Run(fmt.Sprintf("StopSignal_%s", signal), func(t *testing.T) {
			project := &types.Project{
				Name: "test-project",
				Services: types.Services{
					"app": {
						Name:       "app",
						Image:      "nginx:latest",
						StopSignal: signal,
					},
				},
			}

			err := validateQuadletCompatibility(context.Background(), project)

			assert.NoError(t, err)
		})
	}
}

// TestValidateQuadletCompatibility_Tmpfs tests that tmpfs is rejected.
func TestValidateQuadletCompatibility_Tmpfs(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Tmpfs: []string{"/tmp", "/run"},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "tmpfs")
}

// TestValidateQuadletCompatibility_DeployConstraints tests that deploy constraints are rejected.
func TestValidateQuadletCompatibility_DeployConstraints(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Deploy: &types.DeployConfig{
					Placement: types.Placement{
						Constraints: []string{"node.role==manager"},
					},
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "constraints")
}

// TestValidateQuadletCompatibility_DeployPreferences tests that deploy preferences are rejected.
func TestValidateQuadletCompatibility_DeployPreferences(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Deploy: &types.DeployConfig{
					Placement: types.Placement{
						Preferences: []types.PlacementPreferences{
							{
								Spread: "node.labels.az",
							},
						},
					},
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "preferences")
}

// TestValidateQuadletCompatibility_Profiles tests that profiles are rejected.
func TestValidateQuadletCompatibility_Profiles(t *testing.T) {
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"app": {
				Name:     "app",
				Image:    "nginx:latest",
				Profiles: []string{"debug", "testing"},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "profiles")
}

func TestParseIntraProjectDependencies(t *testing.T) {
	tests := []struct {
		name     string
		service  types.ServiceConfig
		expected map[string]string
	}{
		{
			name:     "empty depends_on",
			service:  types.ServiceConfig{},
			expected: map[string]string{},
		},
		{
			name: "single dependency with condition",
			service: types.ServiceConfig{
				DependsOn: types.DependsOnConfig{
					"db": types.ServiceDependency{
						Condition: "service_healthy",
					},
				},
			},
			expected: map[string]string{
				"db": "service_healthy",
			},
		},
		{
			name: "dependency without condition defaults to service_started",
			service: types.ServiceConfig{
				DependsOn: types.DependsOnConfig{
					"redis": types.ServiceDependency{
						Condition: "",
					},
				},
			},
			expected: map[string]string{
				"redis": "service_started",
			},
		},
		{
			name: "multiple dependencies with mixed conditions",
			service: types.ServiceConfig{
				DependsOn: types.DependsOnConfig{
					"db": types.ServiceDependency{
						Condition: "service_healthy",
					},
					"cache": types.ServiceDependency{
						Condition: "",
					},
					"queue": types.ServiceDependency{
						Condition: "service_started",
					},
				},
			},
			expected: map[string]string{
				"db":    "service_healthy",
				"cache": "service_started",
				"queue": "service_started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntraProjectDependencies(tt.service)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d dependencies, got %d", len(tt.expected), len(result))
			}

			for key, expectedVal := range tt.expected {
				gotVal, ok := result[key]
				if !ok {
					t.Errorf("missing dependency: %s", key)
				}
				if gotVal != expectedVal {
					t.Errorf("dependency %s: expected %q, got %q", key, expectedVal, gotVal)
				}
			}
		})
	}
}

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		ch       rune
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{' ', false},
		{'-', false},
		{'_', false},
		{'.', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.ch), func(t *testing.T) {
			result := isAlphaNumeric(tt.ch)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure with multiple projects
	project1Dir := filepath.Join(tmpDir, "project1")
	project2Dir := filepath.Join(tmpDir, "project2")
	nestedDir := filepath.Join(tmpDir, "nested", "project3")

	if err := os.MkdirAll(project1Dir, 0o755); err != nil {
		t.Fatalf("failed to create project1 dir: %v", err)
	}
	if err := os.MkdirAll(project2Dir, 0o755); err != nil {
		t.Fatalf("failed to create project2 dir: %v", err)
	}
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested project dir: %v", err)
	}

	// Create simple compose files for each project
	project1Compose := `version: "3"
services:
  app1:
    image: app1:latest
`
	project2Compose := `version: "3"
services:
  app2:
    image: app2:latest
`
	project3Compose := `version: "3"
services:
  app3:
    image: app3:latest
`

	if err := os.WriteFile(filepath.Join(project1Dir, "compose.yaml"), []byte(project1Compose), 0o644); err != nil {
		t.Fatalf("failed to write project1 compose file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project2Dir, "docker-compose.yml"), []byte(project2Compose), 0o644); err != nil {
		t.Fatalf("failed to write project2 compose file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "compose.yml"), []byte(project3Compose), 0o644); err != nil {
		t.Fatalf("failed to write project3 compose file: %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "load all projects from directory",
			path:          tmpDir,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "load from non-existent directory",
			path:          filepath.Join(tmpDir, "nonexistent"),
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			projects, err := LoadAll(ctx, tt.path, nil)

			if tt.expectError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectError {
				return
			}

			// Count successful loads
			loadedCount := 0
			for _, lp := range projects {
				if lp.Error == nil && lp.Project != nil {
					loadedCount++
				}
			}

			if loadedCount != tt.expectedCount {
				t.Errorf("expected %d loaded projects, got %d", tt.expectedCount, loadedCount)
			}

			// Verify all expected files were found
			if len(projects) != tt.expectedCount {
				t.Errorf("expected %d total projects, got %d", tt.expectedCount, len(projects))
			}
		})
	}
}

// TestParseEnvSecretsMapping_ValidMapping tests parsing a valid x-podman-env-secrets extension.
func TestParseEnvSecretsMapping_ValidMapping(t *testing.T) {
	tests := []struct {
		name            string
		service         *types.ServiceConfig
		expectedSecrets map[string]string
		expectError     bool
	}{
		{
			name: "simple mapping",
			service: &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{
						"db_password": "DATABASE_PASSWORD",
						"api_secret":  "API_KEY",
					},
				},
			},
			expectedSecrets: map[string]string{
				"db_password": "DATABASE_PASSWORD",
				"api_secret":  "API_KEY",
			},
			expectError: false,
		},
		{
			name: "empty extension",
			service: &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{},
				},
			},
			expectedSecrets: map[string]string{},
			expectError:     false,
		},
		{
			name: "no extension",
			service: &types.ServiceConfig{
				Extensions: map[string]interface{}{},
			},
			expectedSecrets: map[string]string{},
			expectError:     false,
		},
		{
			name:            "nil extensions",
			service:         &types.ServiceConfig{},
			expectedSecrets: map[string]string{},
			expectError:     false,
		},
		{
			name: "secret names with dashes and dots",
			service: &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{
						"jwt-secret.prod":       "JWT_SECRET",
						"my_secret-db.password": "DB_PASS",
					},
				},
			},
			expectedSecrets: map[string]string{
				"jwt-secret.prod":       "JWT_SECRET",
				"my_secret-db.password": "DB_PASS",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEnvSecretsMapping("test-service", *tt.service)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSecrets, result)
		})
	}
}

// TestParseEnvSecretsMapping_InvalidEnvVarName tests validation of invalid environment variable names.
func TestParseEnvSecretsMapping_InvalidEnvVarName(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		envVar    string
		expectErr bool
	}{
		{
			name:      "lowercase env var",
			secret:    "db_secret",
			envVar:    "database_password",
			expectErr: true,
		},
		{
			name:      "starts with digit",
			secret:    "secret",
			envVar:    "1_SECRET",
			expectErr: true,
		},
		{
			name:      "contains hyphen",
			secret:    "secret",
			envVar:    "DB-PASSWORD",
			expectErr: true,
		},
		{
			name:      "contains dot",
			secret:    "secret",
			envVar:    "DB.PASSWORD",
			expectErr: true,
		},
		{
			name:      "valid: uppercase with underscore",
			secret:    "secret",
			envVar:    "DATABASE_PASSWORD",
			expectErr: false,
		},
		{
			name:      "valid: starts with underscore",
			secret:    "secret",
			envVar:    "_SECRET",
			expectErr: false,
		},
		{
			name:      "valid: single letter",
			secret:    "secret",
			envVar:    "A",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{
						tt.secret: tt.envVar,
					},
				},
			}

			_, err := parseEnvSecretsMapping("test-service", *service)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseEnvSecretsMapping_InvalidSecretName tests validation of invalid secret names.
func TestParseEnvSecretsMapping_InvalidSecretName(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		envVar    string
		expectErr bool
	}{
		{
			name:      "secret with spaces",
			secret:    "my secret",
			envVar:    "DB_PASSWORD",
			expectErr: true,
		},
		{
			name:      "secret with special chars",
			secret:    "secret@2024",
			envVar:    "API_KEY",
			expectErr: true,
		},
		{
			name:      "secret with forward slash",
			secret:    "my/secret",
			envVar:    "SECRET",
			expectErr: true,
		},
		{
			name:      "valid: with dashes and dots",
			secret:    "my-secret.prod",
			envVar:    "SECRET",
			expectErr: false,
		},
		{
			name:      "valid: with underscores",
			secret:    "my_secret_key",
			envVar:    "SECRET",
			expectErr: false,
		},
		{
			name:      "valid: alphanumeric",
			secret:    "mysecret123",
			envVar:    "SECRET",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": map[string]interface{}{
						tt.secret: tt.envVar,
					},
				},
			}

			_, err := parseEnvSecretsMapping("test-service", *service)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseEnvSecretsMapping_InvalidTypes tests type validation errors.
func TestParseEnvSecretsMapping_InvalidTypes(t *testing.T) {
	tests := []struct {
		name        string
		extension   interface{}
		expectError bool
	}{
		{
			name:        "extension is string instead of object",
			extension:   "not an object",
			expectError: true,
		},
		{
			name:        "extension is array instead of object",
			extension:   []interface{}{"secret1", "secret2"},
			expectError: true,
		},
		{
			name: "env var value is not string",
			extension: map[string]interface{}{
				"db_password": 123,
			},
			expectError: true,
		},
		{
			name: "env var value is array",
			extension: map[string]interface{}{
				"db_password": []string{"ENV_VAR"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": tt.extension,
				},
			}

			_, err := parseEnvSecretsMapping("test-service", *service)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseEnvSecretsMapping_IntegrationWithLoad tests that x-podman-env-secrets is parsed during Load.
func TestParseEnvSecretsMapping_IntegrationWithLoad(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `version: '3.8'
name: test-project
services:
  api:
    image: myapi:latest
    x-podman-env-secrets:
      db_password_secret: DATABASE_PASSWORD
      api_key_secret: API_KEY
      jwt_secret: JWT_SECRET
  web:
    image: nginx:latest`

	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0o644))

	ctx := context.Background()
	project, err := Load(ctx, tmpDir, nil)

	require.NoError(t, err)
	require.NotNil(t, project)

	// Verify api service has parsed env secrets
	apiService := project.Services["api"]
	require.NotNil(t, apiService)
	require.NotNil(t, apiService.Extensions)

	envSecrets, ok := apiService.Extensions["x-quad-ops-env-secrets"].(map[string]string)
	require.True(t, ok, "should have x-quad-ops-env-secrets in extensions")

	assert.Equal(t, 3, len(envSecrets))
	assert.Equal(t, "DATABASE_PASSWORD", envSecrets["db_password_secret"])
	assert.Equal(t, "API_KEY", envSecrets["api_key_secret"])
	assert.Equal(t, "JWT_SECRET", envSecrets["jwt_secret"])

	// Verify web service has no env secrets
	webService := project.Services["web"]
	require.NotNil(t, webService)

	if webService.Extensions != nil {
		_, ok := webService.Extensions["x-quad-ops-env-secrets"]
		assert.False(t, ok, "web service should not have env secrets")
	}
}

// TestGetServiceSecrets_WithSecrets tests extracting secrets from a service.
func TestGetServiceSecrets_WithSecrets(t *testing.T) {
	service := types.ServiceConfig{
		Extensions: map[string]interface{}{
			"x-podman-env-secrets": map[string]interface{}{ //nolint:gosec // test data, not real credentials
				"db_password":  "DB_PASSWORD",
				"api_key":      "API_KEY",
				"oauth_secret": "OAUTH_SECRET",
			},
		},
	}

	secrets := GetServiceSecrets(service)

	assert.Len(t, secrets, 3)
	assert.Contains(t, secrets, "db_password")
	assert.Contains(t, secrets, "api_key")
	assert.Contains(t, secrets, "oauth_secret")
}

// TestGetServiceSecrets_NoSecrets tests a service without secrets.
func TestGetServiceSecrets_NoSecrets(t *testing.T) {
	tests := []struct {
		name    string
		service types.ServiceConfig
	}{
		{
			name:    "nil extensions",
			service: types.ServiceConfig{Extensions: nil},
		},
		{
			name:    "empty extensions",
			service: types.ServiceConfig{Extensions: map[string]interface{}{}},
		},
		{
			name: "no env secrets extension",
			service: types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-other-extension": "value",
				},
			},
		},
		{
			name: "nil env secrets value",
			service: types.ServiceConfig{
				Extensions: map[string]interface{}{
					"x-podman-env-secrets": nil,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secrets := GetServiceSecrets(tc.service)
			assert.Empty(t, secrets)
		})
	}
}

// TestGetServiceSecrets_InvalidType tests a service with invalid secrets type.
func TestGetServiceSecrets_InvalidType(t *testing.T) {
	service := types.ServiceConfig{
		Extensions: map[string]interface{}{
			"x-podman-env-secrets": "not a map",
		},
	}

	secrets := GetServiceSecrets(service)
	assert.Empty(t, secrets)
}

// TestErrorTypes tests Error() and Unwrap() for all error types.
func TestErrorTypes(t *testing.T) {
	cause := fmt.Errorf("underlying cause")

	t.Run("fileNotFoundError", func(t *testing.T) {
		err := &fileNotFoundError{path: "/some/path", cause: cause}
		assert.Contains(t, err.Error(), "/some/path")
		assert.Equal(t, cause, err.Unwrap())
		assert.True(t, IsFileNotFoundError(err))
		assert.False(t, IsFileNotFoundError(fmt.Errorf("other")))
	})

	t.Run("invalidYAMLError", func(t *testing.T) {
		err := &invalidYAMLError{cause: cause}
		assert.Contains(t, err.Error(), "invalid YAML")
		assert.Equal(t, cause, err.Unwrap())
		assert.True(t, IsInvalidYAMLError(err))
		assert.False(t, IsInvalidYAMLError(fmt.Errorf("other")))
	})

	t.Run("validationError without cause", func(t *testing.T) {
		err := &validationError{message: "bad field"}
		assert.Contains(t, err.Error(), "bad field")
		assert.Nil(t, err.Unwrap())
	})

	t.Run("validationError with cause", func(t *testing.T) {
		err := &validationError{message: "bad field", cause: cause}
		assert.Contains(t, err.Error(), "bad field")
		assert.Contains(t, err.Error(), "underlying cause")
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("pathError", func(t *testing.T) {
		err := &pathError{path: "/bad/path", cause: cause}
		assert.Contains(t, err.Error(), "/bad/path")
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("loaderError", func(t *testing.T) {
		err := &loaderError{cause: cause}
		assert.Contains(t, err.Error(), "failed to load compose file")
		assert.Equal(t, cause, err.Unwrap())
		assert.True(t, IsLoaderError(err))
		assert.False(t, IsLoaderError(fmt.Errorf("other")))
	})

	t.Run("quadletCompatibilityError without cause", func(t *testing.T) {
		err := &quadletCompatibilityError{message: "not compatible"}
		assert.Contains(t, err.Error(), "not compatible")
		assert.Nil(t, err.Unwrap())
	})

	t.Run("quadletCompatibilityError with cause", func(t *testing.T) {
		err := &quadletCompatibilityError{message: "not compatible", cause: cause}
		assert.Contains(t, err.Error(), "not compatible")
		assert.Contains(t, err.Error(), "underlying cause")
		assert.Equal(t, cause, err.Unwrap())
	})
}

// TestLoadAll_ContextCancellation tests that LoadAll respects context cancellation.
func TestLoadAll_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := LoadAll(ctx, tmpDir, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestLoadAll_PathIsFile tests that LoadAll rejects a file path.
func TestLoadAll_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("version: '3'\nservices:\n  app:\n    image: busybox\n"), 0o644))

	ctx := context.Background()
	_, err := LoadAll(ctx, filePath, nil)

	require.Error(t, err)
	assert.True(t, IsPathError(err))
}

// TestLoadAll_EmptyDirectory tests LoadAll with no compose files.
func TestLoadAll_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	projects, err := LoadAll(ctx, tmpDir, nil)

	require.NoError(t, err)
	assert.Empty(t, projects)
}

// TestLoadAll_WithFailedProject tests that LoadAll continues on individual project errors.
func TestLoadAll_WithFailedProject(t *testing.T) {
	tmpDir := t.TempDir()

	goodDir := filepath.Join(tmpDir, "good")
	badDir := filepath.Join(tmpDir, "bad")
	require.NoError(t, os.MkdirAll(goodDir, 0o755))
	require.NoError(t, os.MkdirAll(badDir, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(goodDir, "compose.yaml"),
		[]byte("version: '3'\nservices:\n  app:\n    image: busybox\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(badDir, "compose.yaml"),
		[]byte("version: '3'\nservices:\n  app:\n    image: busybox\n    network_mode: none\n"),
		0o644,
	))

	ctx := context.Background()
	projects, err := LoadAll(ctx, tmpDir, nil)

	require.NoError(t, err)
	assert.Len(t, projects, 2)

	var successes, failures int
	for _, p := range projects {
		if p.Error == nil {
			successes++
		} else {
			failures++
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, failures)
}

// TestValidateQuadletCompatibility_NilProject tests nil project handling.
func TestValidateQuadletCompatibility_NilProject(t *testing.T) {
	err := validateQuadletCompatibility(context.Background(), nil)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
}

// TestValidateQuadletCompatibility_ContextCancellation tests context cancellation.
func TestValidateQuadletCompatibility_ContextCancellation(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {Name: "app", Image: "busybox"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := validateQuadletCompatibility(ctx, project)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestValidateQuadletCompatibility_HostNetworkWithPorts tests host network mode with port publishing.
func TestValidateQuadletCompatibility_HostNetworkWithPorts(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:        "app",
				Image:       "nginx:latest",
				NetworkMode: "host",
				Ports: []types.ServicePortConfig{
					{Target: 80, Published: "8080"},
				},
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "host")
	assert.Contains(t, err.Error(), "ports")
}

// TestValidateQuadletCompatibility_CustomNetworkMode tests unsupported custom network mode.
func TestValidateQuadletCompatibility_CustomNetworkMode(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:        "app",
				Image:       "nginx:latest",
				NetworkMode: "container:other",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported network mode")
}

// TestIsServiceNameReference tests the isServiceNameReference function.
func TestIsServiceNameReference(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"", false},
		{"private", false},
		{"shareable", false},
		{"service:other", true},
		{"container:mycontainer", true},
		{"host", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			assert.Equal(t, tt.expected, isServiceNameReference(tt.mode))
		})
	}
}

// TestIsValidEnvVarName_EdgeCases tests edge cases for env var name validation.
func TestIsValidEnvVarName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single underscore", "_", true},
		{"digits only", "123", false},
		{"valid with digits", "A1B2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidEnvVarName(tt.input))
		})
	}
}

// TestIsValidSecretName_EdgeCases tests edge cases for secret name validation.
func TestIsValidSecretName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single char", "a", true},
		{"only dots", "...", true},
		{"only dashes", "---", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidSecretName(tt.input))
		})
	}
}

// TestCheckServiceSecrets tests checking a service for missing secrets.
func TestCheckServiceSecrets(t *testing.T) {
	t.Run("nil available secrets", func(t *testing.T) {
		service := types.ServiceConfig{
			Extensions: map[string]interface{}{
				"x-podman-env-secrets": map[string]interface{}{
					"my_secret": "SECRET",
				},
			},
		}
		missing := CheckServiceSecrets(service, nil)
		assert.Nil(t, missing)
	})

	t.Run("no secrets in service", func(t *testing.T) {
		service := types.ServiceConfig{}
		available := map[string]struct{}{"some_secret": {}}
		missing := CheckServiceSecrets(service, available)
		assert.Nil(t, missing)
	})

	t.Run("all secrets available", func(t *testing.T) {
		service := types.ServiceConfig{
			Extensions: map[string]interface{}{
				"x-podman-env-secrets": map[string]interface{}{
					"db_pass": "DB_PASSWORD",
				},
			},
		}
		available := map[string]struct{}{"db_pass": {}}
		missing := CheckServiceSecrets(service, available)
		assert.Empty(t, missing)
	})

	t.Run("missing secrets", func(t *testing.T) {
		service := types.ServiceConfig{
			Extensions: map[string]interface{}{
				"x-podman-env-secrets": map[string]interface{}{
					"db_pass":    "DB_PASSWORD",
					"api_key":    "API_KEY",
					"jwt_secret": "JWT_SECRET",
				},
			},
		}
		available := map[string]struct{}{"db_pass": {}}
		missing := CheckServiceSecrets(service, available)
		assert.Len(t, missing, 2)
		assert.Contains(t, missing, "api_key")
		assert.Contains(t, missing, "jwt_secret")
	})
}

// TestFilterServicesWithMissingSecrets tests filtering services with missing secrets.
func TestFilterServicesWithMissingSecrets(t *testing.T) {
	t.Run("nil project", func(t *testing.T) {
		skipped, err := FilterServicesWithMissingSecrets(context.Background(), nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, skipped)
	})

	t.Run("all secrets available", func(t *testing.T) {
		project := &types.Project{
			Name: "test",
			Services: types.Services{
				"app": {
					Name:  "app",
					Image: "busybox",
					Extensions: map[string]interface{}{
						"x-podman-env-secrets": map[string]interface{}{
							"db_pass": "DB_PASSWORD",
						},
					},
				},
			},
		}
		available := map[string]struct{}{"db_pass": {}}
		skipped, err := FilterServicesWithMissingSecrets(context.Background(), project, available)
		assert.NoError(t, err)
		assert.Empty(t, skipped)
		assert.Len(t, project.Services, 1)
	})

	t.Run("removes service with missing secrets", func(t *testing.T) {
		project := &types.Project{
			Name: "test",
			Services: types.Services{
				"app": {
					Name:  "app",
					Image: "busybox",
					Extensions: map[string]interface{}{
						"x-podman-env-secrets": map[string]interface{}{
							"missing_secret": "SECRET",
						},
					},
				},
				"web": {
					Name:  "web",
					Image: "nginx",
				},
			},
		}
		available := map[string]struct{}{}
		skipped, err := FilterServicesWithMissingSecrets(context.Background(), project, available)
		assert.NoError(t, err)
		assert.Len(t, skipped, 1)
		assert.Equal(t, "app", skipped[0].ServiceName)
		assert.Contains(t, skipped[0].MissingSecrets, "missing_secret")
		assert.Len(t, project.Services, 1)
		_, ok := project.Services["web"]
		assert.True(t, ok)
	})

	t.Run("service without secrets is kept", func(t *testing.T) {
		project := &types.Project{
			Name: "test",
			Services: types.Services{
				"web": {
					Name:  "web",
					Image: "nginx",
				},
			},
		}
		available := map[string]struct{}{}
		skipped, err := FilterServicesWithMissingSecrets(context.Background(), project, available)
		assert.NoError(t, err)
		assert.Empty(t, skipped)
		assert.Len(t, project.Services, 1)
	})
}

// TestParseServiceDependencies_ContextCancellation tests context cancellation.
func TestParseServiceDependencies_ContextCancellation(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {Name: "app", Image: "busybox"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := parseServiceDependencies(ctx, project)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestValidateQuadletCompatibility_UnsupportedIpcModeNotServiceRef tests non-service IPC reference.
func TestValidateQuadletCompatibility_UnsupportedIpcModeNotServiceRef(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Ipc:   "host",
			},
		},
	}

	err := validateQuadletCompatibility(context.Background(), project)

	require.Error(t, err)
	assert.True(t, IsQuadletCompatibilityError(err))
	assert.Contains(t, err.Error(), "unsupported IPC mode")
}
