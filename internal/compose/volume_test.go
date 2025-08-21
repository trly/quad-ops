package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestVolumeProcessingIntegration(t *testing.T) {
	t.Run("volume processing with various configurations", func(t *testing.T) {
		// Create a processor with mocks
		mockRepo := new(MockRepository)
		mockSystemd := new(MockSystemdManager)
		mockFS := new(MockFileSystem)

		logger := testutil.NewTestLogger(t)
		processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

		project := &types.Project{
			Name: "test-app",
			Volumes: map[string]types.VolumeConfig{
				"data": {
					Name:   "data",
					Driver: "local",
				},
				"logs": {
					Name:   "logs",
					Driver: "local",
					Labels: map[string]string{
						"backup": "daily",
					},
				},
				"external-vol": {
					Name:     "external-vol",
					External: types.External(true),
				},
			},
		}

		// Mock the file system operations that will be called
		mockFS.On("HasUnitChanged", "/Users/trly/.local/share/containers/systemd/test-app-data.volume",
			"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-app-data\n\n[Service]\n").Return(true).Maybe()
		mockFS.On("WriteUnitFile", "/Users/trly/.local/share/containers/systemd/test-app-data.volume",
			"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-app-data\n\n[Service]\n").Return(nil).Maybe()
		mockFS.On("GetUnitFilePath", "test-app-data", "volume").Return("/Users/trly/.local/share/containers/systemd/test-app-data.volume").Maybe()
		mockFS.On("GetContentHash", "[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-app-data\n\n[Service]\n").Return("hash123").Maybe()

		mockFS.On("HasUnitChanged", "/Users/trly/.local/share/containers/systemd/test-app-logs.volume",
			"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nLabel=backup=daily\nVolumeName=test-app-logs\n\n[Service]\n").Return(true).Maybe()
		mockFS.On("WriteUnitFile", "/Users/trly/.local/share/containers/systemd/test-app-logs.volume",
			"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nLabel=backup=daily\nVolumeName=test-app-logs\n\n[Service]\n").Return(nil).Maybe()
		mockFS.On("GetUnitFilePath", "test-app-logs", "volume").Return("/Users/trly/.local/share/containers/systemd/test-app-logs.volume").Maybe()
		mockFS.On("GetContentHash", "[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nLabel=backup=daily\nVolumeName=test-app-logs\n\n[Service]\n").Return("hash456").Maybe()

		// Mock repository calls
		mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()
		mockRepo.On("Create", &repository.Unit{Name: "test-app-data", Type: "volume", SHA1Hash: []byte("hash123")}).Return(
			&repository.Unit{ID: 1, Name: "test-app-data", Type: "volume"}, nil).Maybe()
		mockRepo.On("Create", &repository.Unit{Name: "test-app-logs", Type: "volume", SHA1Hash: []byte("hash456")}).Return(
			&repository.Unit{ID: 2, Name: "test-app-logs", Type: "volume"}, nil).Maybe()

		// Test that processVolumes can be called
		err := processor.processVolumes(project)
		assert.NoError(t, err)

		// Verify that units were processed (should have 2 since external is skipped)
		processedUnits := processor.GetProcessedUnits()
		assert.Len(t, processedUnits, 2)
		assert.True(t, processedUnits["test-app-data.volume"])
		assert.True(t, processedUnits["test-app-logs.volume"])
		assert.False(t, processedUnits["test-app-external-vol.volume"]) // Should be false as external volumes are skipped
	})
}

func TestVolumeExternalBehavior(t *testing.T) {
	t.Run("verify external volumes are skipped", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockSystemd := new(MockSystemdManager)
		mockFS := new(MockFileSystem)

		logger := testutil.NewTestLogger(t)
		processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

		project := &types.Project{
			Name: "test",
			Volumes: map[string]types.VolumeConfig{
				"external-only": {
					Name:     "external-only",
					External: types.External(true),
				},
			},
		}

		// No mocks should be called for external volumes
		err := processor.processVolumes(project)
		assert.NoError(t, err)

		// No units should be processed
		processedUnits := processor.GetProcessedUnits()
		assert.Empty(t, processedUnits)

		// Verify no expectations were missed (no calls should have been made)
		mockRepo.AssertExpectations(t)
		mockSystemd.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})
}

func TestVolumeDriverTypes(t *testing.T) {
	tests := []struct {
		name   string
		driver string
	}{
		{"local driver", "local"},
		{"nfs driver", "nfs"},
		{"tmpfs driver", "tmpfs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockSystemd := new(MockSystemdManager)
			mockFS := new(MockFileSystem)

			logger := testutil.NewTestLogger(t)
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			project := &types.Project{
				Name: "test",
				Volumes: map[string]types.VolumeConfig{
					"test-vol": {
						Name:   "test-vol",
						Driver: tt.driver,
					},
				},
			}

			// Mock the basic operations
			mockFS.On("HasUnitChanged", "/Users/trly/.local/share/containers/systemd/test-test-vol.volume",
				"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-test-vol\n\n[Service]\n").Return(true).Maybe()
			mockFS.On("WriteUnitFile", "/Users/trly/.local/share/containers/systemd/test-test-vol.volume",
				"[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-test-vol\n\n[Service]\n").Return(nil).Maybe()
			mockFS.On("GetUnitFilePath", "test-test-vol", "volume").Return("/Users/trly/.local/share/containers/systemd/test-test-vol.volume").Maybe()
			mockFS.On("GetContentHash", "[Unit]\n\n[Volume]\nLabel=managed-by=quad-ops\nVolumeName=test-test-vol\n\n[Service]\n").Return("hash123").Maybe()

			mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()
			mockRepo.On("Create", &repository.Unit{Name: "test-test-vol", Type: "volume", SHA1Hash: []byte("hash123")}).Return(
				&repository.Unit{ID: 1, Name: "test-test-vol", Type: "volume"}, nil).Maybe()

			err := processor.processVolumes(project)
			assert.NoError(t, err)

			// Verify volume was processed
			processedUnits := processor.GetProcessedUnits()
			assert.Len(t, processedUnits, 1)
			assert.True(t, processedUnits["test-test-vol.volume"])
		})
	}
}
