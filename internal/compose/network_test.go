package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/repository"
)

func TestNetworkProcessingIntegration(t *testing.T) {
	t.Run("network processing with basic configurations", func(t *testing.T) {
		// Create a processor with mocks
		mockRepo := new(MockRepository)
		mockSystemd := new(MockSystemdManager)
		mockFS := new(MockFileSystem)

		logger := initTestLogger()
		processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

		project := &types.Project{
			Name: "test-app",
			Networks: map[string]types.NetworkConfig{
				"frontend": {
					Name:   "frontend",
					Driver: "bridge",
				},
				"external-net": {
					Name:     "external-net",
					External: types.External(true),
				},
			},
		}

		// Mock the file system operations that will be called
		mockFS.On("HasUnitChanged", "/Users/trly/.local/share/containers/systemd/test-app-frontend.network",
			"[Unit]\n\n[Network]\nLabel=managed-by=quad-ops\nNetworkName=test-app-frontend\nDriver=bridge\n\n[Service]\n").Return(true).Maybe()
		mockFS.On("WriteUnitFile", "/Users/trly/.local/share/containers/systemd/test-app-frontend.network",
			"[Unit]\n\n[Network]\nLabel=managed-by=quad-ops\nNetworkName=test-app-frontend\nDriver=bridge\n\n[Service]\n").Return(nil).Maybe()
		mockFS.On("GetUnitFilePath", "test-app-frontend", "network").Return("/Users/trly/.local/share/containers/systemd/test-app-frontend.network").Maybe()
		mockFS.On("GetContentHash", "[Unit]\n\n[Network]\nLabel=managed-by=quad-ops\nNetworkName=test-app-frontend\nDriver=bridge\n\n[Service]\n").Return("hash123").Maybe()

		// Mock repository calls
		mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()
		mockRepo.On("Create", &repository.Unit{Name: "test-app-frontend", Type: "network", SHA1Hash: []byte("hash123")}).Return(
			&repository.Unit{ID: 1, Name: "test-app-frontend", Type: "network"}, nil).Maybe()

		// Test that processNetworks can be called (even if it's a private method,
		// we test through the public interface)
		err := processor.processNetworks(project)
		assert.NoError(t, err)

		// Verify that units were processed (should have 1 since external is skipped)
		processedUnits := processor.GetProcessedUnits()
		assert.Len(t, processedUnits, 1)
		assert.True(t, processedUnits["test-app-frontend.network"])
		assert.False(t, processedUnits["test-app-external-net.network"]) // Should be false as external networks are skipped
	})
}

func TestNetworkExternalBehavior(t *testing.T) {
	t.Run("verify external networks are skipped", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockSystemd := new(MockSystemdManager)
		mockFS := new(MockFileSystem)

		logger := initTestLogger()
		processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

		project := &types.Project{
			Name: "test",
			Networks: map[string]types.NetworkConfig{
				"external-only": {
					Name:     "external-only",
					External: types.External(true),
				},
			},
		}

		// No mocks should be called for external networks
		err := processor.processNetworks(project)
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

func TestNetworkDriverTypes(t *testing.T) {
	tests := []struct {
		name   string
		driver string
	}{
		{"bridge driver", "bridge"},
		{"macvlan driver", "macvlan"},
		{"host driver", "host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockSystemd := new(MockSystemdManager)
			mockFS := new(MockFileSystem)

			logger := initTestLogger()
			processor := NewProcessor(mockRepo, mockSystemd, mockFS, logger, false)

			project := &types.Project{
				Name: "test",
				Networks: map[string]types.NetworkConfig{
					"test-net": {
						Name:   "test-net",
						Driver: tt.driver,
					},
				},
			}

			// Mock the basic operations - include driver in expected content
			expectedContent := "[Unit]\n\n[Network]\nLabel=managed-by=quad-ops\nNetworkName=test-test-net\nDriver=" + tt.driver + "\n\n[Service]\n"
			mockFS.On("HasUnitChanged", "/Users/trly/.local/share/containers/systemd/test-test-net.network",
				expectedContent).Return(true).Maybe()
			mockFS.On("WriteUnitFile", "/Users/trly/.local/share/containers/systemd/test-test-net.network",
				expectedContent).Return(nil).Maybe()
			mockFS.On("GetUnitFilePath", "test-test-net", "network").Return("/Users/trly/.local/share/containers/systemd/test-test-net.network").Maybe()
			mockFS.On("GetContentHash", expectedContent).Return("hash123").Maybe()

			mockRepo.On("FindAll").Return([]repository.Unit{}, nil).Maybe()
			mockRepo.On("Create", &repository.Unit{Name: "test-test-net", Type: "network", SHA1Hash: []byte("hash123")}).Return(
				&repository.Unit{ID: 1, Name: "test-test-net", Type: "network"}, nil).Maybe()

			err := processor.processNetworks(project)
			assert.NoError(t, err)

			// Verify network was processed
			processedUnits := processor.GetProcessedUnits()
			assert.Len(t, processedUnits, 1)
			assert.True(t, processedUnits["test-test-net.network"])
		})
	}
}
