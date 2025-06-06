package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestFromComposeBuild(t *testing.T) {
	// Basic test case with simple build configuration
	buildConfig := types.BuildConfig{
		Context:    "./app",
		Dockerfile: "Dockerfile",
	}

	service := types.ServiceConfig{
		Name:  "webapp",
		Image: "webapp:latest",
	}

	build := NewBuild("test-webapp-build")
	build = build.FromComposeBuild(buildConfig, service, "test")

	// Verify basic fields are set correctly
	assert.Equal(t, "test-webapp-build", build.Name)
	assert.Equal(t, "build", build.UnitType)
	assert.Equal(t, "Dockerfile", build.File)
	assert.Equal(t, "repo", build.SetWorkingDirectory) // Should default to "repo" for regular path
	assert.Contains(t, build.ImageTag, "webapp:latest")
}

func TestFromComposeBuildWithGitURL(t *testing.T) {
	// Test with Git URL as context
	buildConfig := types.BuildConfig{
		Context:    "https://github.com/example/repo.git",
		Dockerfile: "Dockerfile.prod",
	}

	service := types.ServiceConfig{
		Name:  "api",
		Image: "api:latest",
	}

	build := NewBuild("test-api-build")
	build = build.FromComposeBuild(buildConfig, service, "test")

	// Verify Git URL is handled correctly
	assert.Equal(t, "Dockerfile.prod", build.File)
	assert.Equal(t, "https://github.com/example/repo.git", build.SetWorkingDirectory)
	assert.Contains(t, build.ImageTag, "api:latest")
}

func TestFromComposeBuildWithComplexConfig(t *testing.T) {
	// Test with more complex build configuration including args and labels
	arg1 := "value1"
	arg2 := "value2"
	buildConfig := types.BuildConfig{
		Context:    "./app",
		Dockerfile: "Dockerfile.dev",
		Args: map[string]*string{
			"ARG1": &arg1,
			"ARG2": &arg2,
		},
		Labels: types.Labels{
			"com.example.vendor":  "Example Corp",
			"com.example.version": "1.0",
		},
		Target: "dev",
		Pull:   true,
	}

	service := types.ServiceConfig{
		Name:  "frontend",
		Image: "frontend:dev",
	}

	build := NewBuild("test-frontend-build")
	build = build.FromComposeBuild(buildConfig, service, "test")

	// Verify complex configuration is set correctly
	assert.Equal(t, "Dockerfile.dev", build.File)
	assert.Equal(t, 2, len(build.Env))
	assert.Equal(t, "value1", build.Env["ARG1"])
	assert.Equal(t, "value2", build.Env["ARG2"])
	assert.Contains(t, build.Label, "com.example.vendor=Example Corp")
	assert.Contains(t, build.Label, "com.example.version=1.0")
	assert.Equal(t, "dev", build.Target)
	assert.Equal(t, "always", build.Pull)
}

// import existing initTestLogger from environment_file_test.go

func TestBuildUnitGeneration(t *testing.T) {
	// Initialize test logger
	initTestLogger()

	// Test generating the actual unit file content
	arg1 := "value1"
	buildConfig := types.BuildConfig{
		Context:    "./app",
		Dockerfile: "Dockerfile",
		Args: map[string]*string{
			"ARG1": &arg1,
		},
	}

	service := types.ServiceConfig{
		Name:  "app",
		Image: "app:latest",
	}

	build := NewBuild("test-app-build")
	build = build.FromComposeBuild(buildConfig, service, "test")

	// Create the quadlet unit
	quadletUnit := QuadletUnit{
		Name:  "test-app-build",
		Type:  "build",
		Build: *build,
		Systemd: SystemdConfig{
			RemainAfterExit: true,
		},
	}

	// Generate the unit file content
	content := GenerateQuadletUnit(quadletUnit)

	// Verify the content has the expected sections and keys
	assert.Contains(t, content, "[Unit]")
	assert.Contains(t, content, "[Build]")
	assert.Contains(t, content, "[Service]")
	assert.Contains(t, content, "ImageTag=app:latest")
	assert.Contains(t, content, "File=Dockerfile")
	assert.Contains(t, content, "SetWorkingDirectory=repo")
	assert.Contains(t, content, "Environment=ARG1=value1")
	assert.Contains(t, content, "RemainAfterExit=yes")
	assert.Contains(t, content, "Label=managed-by=quad-ops")
}
