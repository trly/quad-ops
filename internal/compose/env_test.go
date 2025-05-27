package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/log"
)

func TestEnvFileVariableInterpolation(t *testing.T) {
	// Initialize logger for test
	log.Init(true)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "compose-env-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .env file
	envContent := `
UPLOAD_LOCATION=test-library-data
DB_USERNAME=test-user
DB_PASSWORD=test-password
`
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Create docker-compose.yaml file that uses variables from .env
	composeContent := `
services:
  test-service:
    image: test/image:latest
    volumes:
      - ${UPLOAD_LOCATION}:/data
    environment:
      - DB_USER=${DB_USERNAME}
      - DB_PASS=${DB_PASSWORD}

volumes:
  test-library-data: {}
`

	composePath := filepath.Join(tmpDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Parse the compose file
	project, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that variables were interpolated correctly
	assert.Len(t, project.Services, 1)
	assert.Contains(t, project.Services, "test-service")

	// Check volumes with interpolated variable
	assert.Len(t, project.Services["test-service"].Volumes, 1)
	assert.Equal(t, "test-library-data", project.Services["test-service"].Volumes[0].Source)
	assert.Equal(t, "/data", project.Services["test-service"].Volumes[0].Target)

	// Check environment variables
	assert.Contains(t, project.Services["test-service"].Environment, "DB_USER")
	assert.Equal(t, "test-user", *project.Services["test-service"].Environment["DB_USER"])
	assert.Contains(t, project.Services["test-service"].Environment, "DB_PASS")
	assert.Equal(t, "test-password", *project.Services["test-service"].Environment["DB_PASS"])
}

func TestValidateEnvKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectErr bool
	}{
		{"valid uppercase key", "MYAPP_CONFIG", false},
		{"valid mixed case key", "MyApp_Config", false},
		{"valid with numbers", "CONFIG_V2", false},
		{"empty key", "", true},
		{"starts with digit", "2CONFIG", true},
		{"contains spaces", "MY CONFIG", true},
		{"contains special chars", "MY-CONFIG", true},
		{"contains dots", "MY.CONFIG", true},
		{"critical PATH variable", "PATH", true},
		{"critical HOME variable", "HOME", true},
		{"critical USER variable", "USER", true},
		{"critical SHELL variable", "SHELL", true},
		{"critical PWD variable", "PWD", true},
		{"critical OLDPWD variable", "OLDPWD", true},
		{"critical TERM variable", "TERM", true},
		{"case insensitive critical", "path", true},
		{"case insensitive critical", "Home", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvKey(tt.key)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for key: %s", tt.key)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for key %s: %v", tt.key, err)
			}
		})
	}
}
