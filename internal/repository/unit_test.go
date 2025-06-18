package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
)

func TestSystemdRepository_FindAll(t *testing.T) {
	// Initialize logger to avoid nil pointer
	log.Init(false)

	cfg := &config.Settings{
		Verbose: false,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Setup temporary directory for test unit files
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup config to use temp directory
	cfg = &config.Settings{
		QuadletDir: tempDir,
		UserMode:   false,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Create test unit files
	containerContent := `[Container]
Image=nginx:latest
ContainerName=test-nginx
`
	volumeContent := `[Volume]
VolumeName=test-volume
`

	err = os.WriteFile(filepath.Join(tempDir, "test-nginx.container"), []byte(containerContent), 0600)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test-volume.volume"), []byte(volumeContent), 0600)
	assert.NoError(t, err)

	// Test repository
	repo := NewRepository()
	units, err := repo.FindAll()

	assert.NoError(t, err)
	assert.Len(t, units, 2)

	// Check if we have both units
	foundContainer := false
	foundVolume := false
	for _, unit := range units {
		if unit.Name == "test-nginx" && unit.Type == "container" {
			foundContainer = true
			assert.Equal(t, "keep", unit.CleanupPolicy) // Default policy
			assert.False(t, unit.UserMode)
		}
		if unit.Name == "test-volume" && unit.Type == "volume" {
			foundVolume = true
		}
	}
	assert.True(t, foundContainer, "Container unit should be found")
	assert.True(t, foundVolume, "Volume unit should be found")
}

func TestSystemdRepository_FindByUnitType(t *testing.T) {
	// Setup temporary directory for test unit files
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup config to use temp directory
	cfg := &config.Settings{
		QuadletDir: tempDir,
		UserMode:   false,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Create test unit files of different types
	containerContent := `[Container]
Image=nginx:latest
`
	volumeContent := `[Volume]
VolumeName=test-volume
`

	err = os.WriteFile(filepath.Join(tempDir, "test-nginx.container"), []byte(containerContent), 0600)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test-db.container"), []byte(containerContent), 0600)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "test-volume.volume"), []byte(volumeContent), 0600)
	assert.NoError(t, err)

	// Test repository
	repo := NewRepository()

	// Find only container units
	containerUnits, err := repo.FindByUnitType("container")
	assert.NoError(t, err)
	assert.Len(t, containerUnits, 2)

	// Find only volume units
	volumeUnits, err := repo.FindByUnitType("volume")
	assert.NoError(t, err)
	assert.Len(t, volumeUnits, 1)

	// Find non-existent type
	networkUnits, err := repo.FindByUnitType("network")
	assert.NoError(t, err)
	assert.Len(t, networkUnits, 0)
}

func TestSystemdRepository_Create(t *testing.T) {
	repo := NewRepository()

	// Create a unit (should succeed but not store anything)
	unit := &Unit{
		Name:          "test-unit",
		Type:          "container",
		SHA1Hash:      []byte("abc123"),
		CleanupPolicy: "delete",
		UserMode:      false,
	}

	id, err := repo.Create(unit)
	assert.NoError(t, err)
	assert.Greater(t, id, int64(0)) // Should return a generated ID
}

func TestSystemdRepository_Delete(t *testing.T) {
	repo := NewRepository()

	// Delete a unit (should succeed but not actually delete anything)
	err := repo.Delete(123)
	assert.NoError(t, err)
}

func TestSystemdRepository_FindByID(t *testing.T) {
	// Setup temporary directory for test unit files
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup config to use temp directory
	cfg := &config.Settings{
		QuadletDir: tempDir,
		UserMode:   false,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Create a test unit file
	containerContent := `[Container]
Image=nginx:latest
`
	err = os.WriteFile(filepath.Join(tempDir, "test-nginx.container"), []byte(containerContent), 0600)
	assert.NoError(t, err)

	repo := NewRepository()

	// Get all units to find a valid ID
	units, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Len(t, units, 1)

	testID := units[0].ID

	// Find by ID
	unit, err := repo.FindByID(testID)
	assert.NoError(t, err)
	assert.Equal(t, "test-nginx", unit.Name)
	assert.Equal(t, "container", unit.Type)

	// Find by non-existent ID
	_, err = repo.FindByID(999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestParseUnitFromFile(t *testing.T) {
	// Setup temporary directory for test unit files
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup config to use temp directory
	cfg := &config.Settings{
		QuadletDir: tempDir,
		UserMode:   true,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Create a test unit file
	containerContent := `[Container]
Image=nginx:latest
ContainerName=test-nginx
`
	filePath := filepath.Join(tempDir, "test-nginx.container")
	err = os.WriteFile(filePath, []byte(containerContent), 0600)
	assert.NoError(t, err)

	repo := &SystemdRepository{}
	unit, err := repo.parseUnitFromFile(filePath, "test-nginx", "container")

	assert.NoError(t, err)
	assert.Equal(t, "test-nginx", unit.Name)
	assert.Equal(t, "container", unit.Type)
	assert.Equal(t, "keep", unit.CleanupPolicy) // Default
	assert.True(t, unit.UserMode)               // From config
	assert.NotEmpty(t, unit.SHA1Hash)
	assert.NotZero(t, unit.CreatedAt)
}

func TestHashFunction(t *testing.T) {
	// Test hash function generates consistent values
	hash1 := hash("test-string")
	hash2 := hash("test-string")
	hash3 := hash("different-string")

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}

func TestUserModeHandling(t *testing.T) {
	// Setup temporary directory for test unit files
	tempDir, err := os.MkdirTemp("", "quad-ops-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test with userMode=false
	cfg := &config.Settings{
		QuadletDir: tempDir,
		UserMode:   false,
	}
	config.DefaultProvider().SetConfig(cfg)

	// Create a test unit file
	containerContent := `[Container]
Image=nginx:latest
`
	err = os.WriteFile(filepath.Join(tempDir, "test-nginx.container"), []byte(containerContent), 0600)
	assert.NoError(t, err)

	repo := NewRepository()
	units, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Len(t, units, 1)
	assert.False(t, units[0].UserMode)

	// Switch to userMode=true
	cfg.UserMode = true

	units, err = repo.FindAll()
	assert.NoError(t, err)
	assert.Len(t, units, 1)
	assert.True(t, units[0].UserMode) // Should reflect current config
}

func TestContentHashCalculation(t *testing.T) {
	content1 := "test content"
	content2 := "test content"
	content3 := "different content"

	hash1 := fs.GetContentHash(content1)
	hash2 := fs.GetContentHash(content2)
	hash3 := fs.GetContentHash(content3)

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}
