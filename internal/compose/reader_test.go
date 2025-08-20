package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// The project name is based on the directory containing the temp file
	// We need to determine what the actual expected name should be based on the temp directory
	expectedName := filepath.Base(filepath.Dir(tmpfile.Name()))
	expectedName = strings.ToLower(expectedName) // sanitizeProjectName converts to lowercase
	assert.Equal(t, expectedName, project.Name)
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
func TestReadProjectsEmptyPath(t *testing.T) {
	_, err := ReadProjects("")
	assert.Error(t, err)
}

func TestReadProjectsMissingDirectory(t *testing.T) {
	_, err := ReadProjects("testdata/missing-directory")
	assert.Error(t, err)
}

func TestReadProjectsPermissionDenied(t *testing.T) {
	parentDir, _ := os.MkdirTemp("", "parent")
	testDir := filepath.Join(parentDir, "secureDir")
	_ = os.MkdirAll(testDir, 0755) //#nosec used for testing

	_ = os.Chmod(parentDir, 0644) // #nosec used for testing

	defer func() {
		_ = os.Chmod(parentDir, 0755) // #nosec used for testing
		_ = os.RemoveAll(parentDir)
	}()

	_, err := ReadProjects(testDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access compose directory")
}

func TestReadProjectsWalkDirectoryStructure(t *testing.T) {
	log.Init(true)

	tmpDir, err := os.MkdirTemp("", "compose-walk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	composeContent := `
services:
  test-service:
    image: test/image:latest
    ports:
      - "8080:80"
`

	rootComposePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(rootComposePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	//#nosec used for testing
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	subComposePath := filepath.Join(subDir, "compose.yaml")
	if err := os.WriteFile(subComposePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	nestedDir := filepath.Join(subDir, "nested")
	//#nosec used for testing
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedComposePath := filepath.Join(nestedDir, "docker-compose.yaml")
	if err := os.WriteFile(nestedComposePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	ignoredYamlPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(ignoredYamlPath, []byte("key: value"), 0600); err != nil {
		t.Fatal(err)
	}

	ignoredYamlPath2 := filepath.Join(subDir, "settings.yaml")
	if err := os.WriteFile(ignoredYamlPath2, []byte("setting: value"), 0600); err != nil {
		t.Fatal(err)
	}

	txtFilePath := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFilePath, []byte("This is a readme"), 0600); err != nil {
		t.Fatal(err)
	}

	projects, err := ReadProjects(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, projects, 3, "Should find exactly 3 compose files")

	projectNames := make([]string, len(projects))
	for i, project := range projects {
		projectNames[i] = project.Name
		assert.Len(t, project.Services, 1, "Each project should have 1 service")
		assert.Contains(t, project.Services, "test-service", "Each project should contain test-service")
	}

	expectedRootName := filepath.Base(tmpDir)
	assert.Contains(t, projectNames, expectedRootName, "Should contain root directory project")
	assert.Contains(t, projectNames, "subdir", "Should contain subdir project")
	assert.Contains(t, projectNames, "nested", "Should contain nested project")
}

func TestReadProjectsWalkEmptyDirectory(t *testing.T) {
	log.Init(true)

	tmpDir, err := os.MkdirTemp("", "compose-empty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	projects, err := ReadProjects(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, projects, 0, "Should return empty slice for directory with no compose files")
}

func TestReadProjectsWalkWithDirectoryAccessError(t *testing.T) {
	log.Init(true)

	tmpDir, err := os.MkdirTemp("", "compose-access-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	restrictedDir := filepath.Join(tmpDir, "restricted")
	//#nosec used for testing
	if err := os.MkdirAll(restrictedDir, 0755); err != nil {
		t.Fatal(err)
	}

	composeContent := `services:
  test-service:
    image: test/image:latest`

	mainComposePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(mainComposePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(restrictedDir, 0000); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.Chmod(restrictedDir, 0755) // #nosec used for testing
	}()

	projects, err := ReadProjects(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, projects, 1, "Should find 1 compose file despite subdirectory access error")
	assert.Contains(t, projects[0].Services, "test-service", "Should parse the accessible compose file")
}
func TestReadProjectsNotADirectory(t *testing.T) {
	testFile, _ := os.CreateTemp("", "testfile")
	_, err := ReadProjects(testFile.Name())
	assert.Error(t, err)
}

func TestParseComposeFileProjectNaming(t *testing.T) {
	// Initialize logger for test
	log.Init(true)

	composeContent := `
services:
  test-service:
    image: test/image:latest
`

	tests := []struct {
		name         string
		pathPattern  string
		expectedName string
	}{
		{
			name:         "repositories pattern with nested directory",
			pathPattern:  "repositories/myrepo/webapp/docker-compose.yml",
			expectedName: "myrepo-webapp",
		},
		{
			name:         "repositories pattern with root compose file",
			pathPattern:  "repositories/myapp/docker-compose.yml",
			expectedName: "myapp-myapp",
		},
		{
			name:         "repositories pattern with deeply nested path",
			pathPattern:  "repositories/backend-service/services/api/docker-compose.yml",
			expectedName: "backend-service-api",
		},
		{
			name:         "non-repositories pattern",
			pathPattern:  "projects/myapp/docker-compose.yml",
			expectedName: "myapp",
		},
		{
			name:         "simple directory",
			pathPattern:  "myapp/docker-compose.yml",
			expectedName: "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tmpDir, err := os.MkdirTemp("", "compose-naming-test")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create the full path structure
			fullPath := filepath.Join(tmpDir, tt.pathPattern)
			dirPath := filepath.Dir(fullPath)
			//#nosec used for testing
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatal(err)
			}

			// Create the compose file
			if err := os.WriteFile(fullPath, []byte(composeContent), 0600); err != nil {
				t.Fatal(err)
			}

			// Parse the compose file
			project, err := ParseComposeFile(fullPath)
			if err != nil {
				t.Fatal(err)
			}

			// Verify the project name
			assert.Equal(t, tt.expectedName, project.Name, "Project name should match expected pattern")
		})
	}
}

func TestParseComposeFileRepositoriesEdgeCases(t *testing.T) {
	// Initialize logger for test
	log.Init(true)

	composeContent := `
services:
  test-service:
    image: test/image:latest
`

	tests := []struct {
		name         string
		pathPattern  string
		expectedName string
		description  string
	}{
		{
			name:         "repositories at end of path",
			pathPattern:  "some/path/repositories/docker-compose.yml",
			expectedName: "repositories",
			description:  "When repositories is the directory containing compose file",
		},
		{
			name:         "multiple repositories in path",
			pathPattern:  "repositories/myrepo/repositories/submodule/docker-compose.yml",
			expectedName: "myrepo-submodule",
			description:  "Should use first repositories directory found",
		},
		{
			name:         "repositories without sufficient components",
			pathPattern:  "repositories/docker-compose.yml",
			expectedName: "repositories",
			description:  "When there's no repo name after repositories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tmpDir, err := os.MkdirTemp("", "compose-edge-test")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create the full path structure
			fullPath := filepath.Join(tmpDir, tt.pathPattern)
			dirPath := filepath.Dir(fullPath)
			//#nosec used for testing
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatal(err)
			}

			// Create the compose file
			if err := os.WriteFile(fullPath, []byte(composeContent), 0600); err != nil {
				t.Fatal(err)
			}

			// Parse the compose file
			project, err := ParseComposeFile(fullPath)
			if err != nil {
				t.Fatal(err)
			}

			// Verify the project name
			assert.Equal(t, tt.expectedName, project.Name, tt.description)
		})
	}
}

func TestParseComposeFileDefaultNaming(t *testing.T) {
	// Initialize logger for test
	log.Init(true)

	composeContent := `
services:
  test-service:
    image: test/image:latest
`

	// Test edge case where directory name results in default
	tmpDir, err := os.MkdirTemp("", "compose-default-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a subdirectory that might trigger default naming
	testDir := filepath.Join(tmpDir, ".")
	//#nosec used for testing
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	composePath := filepath.Join(testDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Parse the compose file
	project, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatal(err)
	}

	// Should use parent directory name, not "default"
	assert.NotEqual(t, "default", project.Name, "Should not use default name for normal directories")
	assert.NotEmpty(t, project.Name, "Project name should not be empty")
}
