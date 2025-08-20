package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/repository"
)

func TestPrefix(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		resourceName string
		expected     string
	}{
		{
			name:         "Normal prefix",
			projectName:  "myapp",
			resourceName: "webapp",
			expected:     "myapp-webapp",
		},
		{
			name:         "Empty project name",
			projectName:  "",
			resourceName: "webapp",
			expected:     "-webapp",
		},
		{
			name:         "Empty resource name",
			projectName:  "myapp",
			resourceName: "",
			expected:     "myapp-",
		},
		{
			name:         "Both empty",
			projectName:  "",
			resourceName: "",
			expected:     "-",
		},
		{
			name:         "Project with special chars",
			projectName:  "my-app_v2",
			resourceName: "web.service",
			expected:     "my-app_v2-web.service",
		},
		{
			name:         "Numeric names",
			projectName:  "app1",
			resourceName: "service2",
			expected:     "app1-service2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Prefix(tt.projectName, tt.resourceName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindEnvFiles(t *testing.T) {
	// Create temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "quad-ops-test-env")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name        string
		serviceName string
		workingDir  string
		setupFiles  []string
		expected    []string
	}{
		{
			name:        "Empty working directory",
			serviceName: "webapp",
			workingDir:  "",
			setupFiles:  []string{},
			expected:    nil,
		},
		{
			name:        "No env files exist",
			serviceName: "webapp",
			workingDir:  tmpDir,
			setupFiles:  []string{},
			expected:    nil,
		},
		{
			name:        "Only general .env file",
			serviceName: "webapp",
			workingDir:  tmpDir,
			setupFiles:  []string{".env"},
			expected:    []string{filepath.Join(tmpDir, ".env")},
		},
		{
			name:        "Service-specific .env.service pattern",
			serviceName: "webapp",
			workingDir:  tmpDir,
			setupFiles:  []string{".env", ".env.webapp"},
			expected: []string{
				filepath.Join(tmpDir, ".env"),
				filepath.Join(tmpDir, ".env.webapp"),
			},
		},
		{
			name:        "Service-specific service.env pattern",
			serviceName: "api",
			workingDir:  tmpDir,
			setupFiles:  []string{".env", "api.env"},
			expected: []string{
				filepath.Join(tmpDir, ".env"),
				filepath.Join(tmpDir, "api.env"),
			},
		},
		{
			name:        "Env files in env subdirectory",
			serviceName: "database",
			workingDir:  tmpDir,
			setupFiles:  []string{".env", "env/database.env"},
			expected: []string{
				filepath.Join(tmpDir, ".env"),
				filepath.Join(tmpDir, "env", "database.env"),
			},
		},
		{
			name:        "Env files in envs subdirectory",
			serviceName: "redis",
			workingDir:  tmpDir,
			setupFiles:  []string{".env", "envs/redis.env"},
			expected: []string{
				filepath.Join(tmpDir, ".env"),
				filepath.Join(tmpDir, "envs", "redis.env"),
			},
		},
		{
			name:        "Multiple env file patterns",
			serviceName: "worker",
			workingDir:  tmpDir,
			setupFiles:  []string{".env", ".env.worker", "worker.env", "env/worker.env", "envs/worker.env"},
			expected: []string{
				filepath.Join(tmpDir, ".env"),
				filepath.Join(tmpDir, ".env.worker"),
				filepath.Join(tmpDir, "worker.env"),
				filepath.Join(tmpDir, "env", "worker.env"),
				filepath.Join(tmpDir, "envs", "worker.env"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing files
			_ = os.RemoveAll(filepath.Join(tmpDir, ".env"))
			_ = os.RemoveAll(filepath.Join(tmpDir, ".env.webapp"))
			_ = os.RemoveAll(filepath.Join(tmpDir, ".env.worker"))
			_ = os.RemoveAll(filepath.Join(tmpDir, "api.env"))
			_ = os.RemoveAll(filepath.Join(tmpDir, "worker.env"))
			_ = os.RemoveAll(filepath.Join(tmpDir, "env"))
			_ = os.RemoveAll(filepath.Join(tmpDir, "envs"))

			// Setup test files
			for _, file := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, file)
				dir := filepath.Dir(fullPath)
				if dir != tmpDir {
					err := os.MkdirAll(dir, 0750)
					require.NoError(t, err)
				}
				err := os.WriteFile(fullPath, []byte("TEST=true"), 0600)
				require.NoError(t, err)
			}

			result := FindEnvFiles(tt.serviceName, tt.workingDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRepository for testing HasNamingConflict.
type TestRepository struct {
	units []repository.Unit
	err   error
}

func (t *TestRepository) FindAll() ([]repository.Unit, error) {
	return t.units, t.err
}

func (t *TestRepository) Create(unit *repository.Unit) (*repository.Unit, error) {
	return unit, nil
}

func (t *TestRepository) Delete(_ string) error {
	return nil
}

func TestHasNamingConflict(t *testing.T) {
	tests := []struct {
		name          string
		unitName      string
		unitType      string
		existingUnits []repository.Unit
		repoError     error
		expected      bool
	}{
		{
			name:          "No existing units",
			unitName:      "myapp-webapp",
			unitType:      "container",
			existingUnits: []repository.Unit{},
			expected:      false,
		},
		{
			name:          "Repository error returns false",
			unitName:      "myapp-webapp",
			unitType:      "container",
			existingUnits: []repository.Unit{},
			repoError:     assert.AnError,
			expected:      false,
		},
		{
			name:     "Exact same unit exists - no conflict",
			unitName: "myapp-webapp",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "myapp-webapp", Type: "container"},
			},
			expected: false,
		},
		{
			name:     "Different type - no conflict",
			unitName: "myapp-webapp",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "myapp-webapp", Type: "network"},
			},
			expected: false,
		},
		{
			name:     "Suffix conflict detected",
			unitName: "myapp-webapp",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "webapp", Type: "container"},
			},
			expected: true,
		},
		{
			name:     "Prefix conflict detected",
			unitName: "webapp",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "myapp-webapp", Type: "container"},
			},
			expected: true,
		},
		{
			name:     "Multiple units - one with conflict",
			unitName: "myapp-db",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "myapp-webapp", Type: "container"},
				{Name: "db", Type: "container"},
				{Name: "myapp-redis", Type: "container"},
			},
			expected: true,
		},
		{
			name:     "Multiple units - no conflicts",
			unitName: "myapp-queue",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "myapp-webapp", Type: "container"},
				{Name: "myapp-database", Type: "container"},
				{Name: "myapp-redis", Type: "container"},
			},
			expected: false,
		},
		{
			name:     "Complex naming patterns",
			unitName: "project-service-v2",
			unitType: "container",
			existingUnits: []repository.Unit{
				{Name: "service-v2", Type: "container"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &TestRepository{
				units: tt.existingUnits,
				err:   tt.repoError,
			}

			result := HasNamingConflict(repo, tt.unitName, tt.unitType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsExternal(t *testing.T) {
	tests := []struct {
		name     string
		external interface{}
		expected bool
	}{
		{
			name:     "Nil value",
			external: nil,
			expected: false,
		},
		{
			name:     "Bool true",
			external: true,
			expected: true,
		},
		{
			name:     "Bool false",
			external: false,
			expected: false,
		},
		{
			name:     "Bool pointer true",
			external: func() *bool { b := true; return &b }(),
			expected: true,
		},
		{
			name:     "Bool pointer false",
			external: func() *bool { b := false; return &b }(),
			expected: false,
		},
		{
			name:     "Bool pointer nil",
			external: (*bool)(nil),
			expected: false,
		},
		{
			name:     "String value",
			external: "true",
			expected: false,
		},
		{
			name:     "Integer value",
			external: 1,
			expected: false,
		},
		{
			name:     "Float value",
			external: 1.0,
			expected: false,
		},
		{
			name:     "Slice value",
			external: []string{"external"},
			expected: false,
		},
		{
			name:     "Map value",
			external: map[string]interface{}{"external": true},
			expected: false,
		},
		{
			name:     "Struct value",
			external: struct{ External bool }{External: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExternal(tt.external)
			assert.Equal(t, tt.expected, result)
		})
	}
}
