package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/log"
)

func TestParseComposeFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "docker-compose.yaml")

	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	composeContent := `
services:
  frontend:
    image: example/webapp
    ports:
      - "443:8043"
    networks:
      - front-tier
      - back-tier
    configs:
      - httpd-config
    secrets:
      - server-certificate

  backend:
    image: example/database
    volumes:
      - db-data:/etc/data
    networks:
      - back-tier

volumes:
  db-data:
    driver: flocker
    driver_opts:
      size: "10GiB"

configs:
  httpd-config:
    external: true

secrets:
  server-certificate:
    external: true

networks:
  front-tier: {}
  back-tier: {}
`

	if _, err := tmpfile.WriteString(composeContent); err != nil {
		_ = tmpfile.Close()
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	project, err := ParseComposeFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "tmp", project.Name)
	assert.Len(t, project.Services, 2)
	assert.Equal(t, "frontend", project.Services["frontend"].Name)
	assert.Len(t, project.Services["frontend"].Volumes, 0)
	assert.Equal(t, "example/webapp", project.Services["frontend"].Image)
	assert.Len(t, project.Services["frontend"].Ports, 1)
	assert.Equal(t, types.ServicePortConfig{Mode: "ingress", Published: "443", Target: 8043, Protocol: "tcp"}, project.Services["frontend"].Ports[0])
	assert.Len(t, project.Services["frontend"].Networks, 2)
	assert.Contains(t, project.Services["frontend"].Networks, "front-tier")
	assert.Contains(t, project.Services["frontend"].Networks, "back-tier")
	assert.Len(t, project.Services["frontend"].Configs, 1)

	assert.Len(t, project.Services["frontend"].Secrets, 1)

	assert.Equal(t, "backend", project.Services["backend"].Name)
	assert.Equal(t, "example/database", project.Services["backend"].Image)
	assert.Len(t, project.Services["backend"].Volumes, 1)
	assert.Equal(t, "db-data", project.Services["backend"].Volumes[0].Source)
	assert.Equal(t, "/etc/data", project.Services["backend"].Volumes[0].Target)

	assert.Len(t, project.Services["backend"].Networks, 1)

	assert.Len(t, project.Volumes, 1)
	assert.Equal(t, fmt.Sprintf("%s_db-data", project.Name), project.Volumes["db-data"].Name)
	assert.Equal(t, "flocker", project.Volumes["db-data"].Driver)
	assert.Equal(t, "10GiB", project.Volumes["db-data"].DriverOpts["size"])

	assert.Len(t, project.Configs, 1)
	assert.Equal(t, "httpd-config", project.Configs["httpd-config"].Name)
	assert.Equal(t, types.External(true), project.Configs["httpd-config"].External)

	assert.Len(t, project.Secrets, 1)
	assert.Equal(t, "server-certificate", project.Secrets["server-certificate"].Name)
	assert.Equal(t, types.External(true), project.Secrets["server-certificate"].External)

	assert.Len(t, project.Networks, 2)
	assert.Equal(t, fmt.Sprintf("%s_front-tier", project.Name), project.Networks["front-tier"].Name)
	assert.Equal(t, fmt.Sprintf("%s_back-tier", project.Name), project.Networks["back-tier"].Name)
}

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
