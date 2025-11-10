package systemd

import (
	"context"
	"errors"
	"testing"

	dbusapi "github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/testutil"
)

// MockFileSystem provides mock file system operations for testing diagnostics.
type MockFileSystem struct {
	existingFiles map[string]bool
	fileError     error
}

func (m *MockFileSystem) Stat(path string) (bool, error) {
	if m.fileError != nil {
		return false, m.fileError
	}
	exists, ok := m.existingFiles[path]
	if !ok {
		return false, nil
	}
	return exists, nil
}

// MockConnectionFactory creates mock connections for diagnostics testing.
type MockDiagnosticsConnectionFactory struct {
	loadedUnits map[string]bool
	connError   error
}

func (m *MockDiagnosticsConnectionFactory) NewConnection(_ context.Context, _ bool) (Connection, error) {
	if m.connError != nil {
		return nil, m.connError
	}
	return &MockDiagnosticsConnection{loadedUnits: m.loadedUnits}, nil
}

// MockDiagnosticsConnection simulates systemd D-Bus connection for diagnostics.
type MockDiagnosticsConnection struct {
	loadedUnits map[string]bool
}

func (m *MockDiagnosticsConnection) GetUnitProperties(_ context.Context, unitName string) (map[string]interface{}, error) {
	loaded, exists := m.loadedUnits[unitName]
	if !exists || !loaded {
		return nil, errors.New("unit not found")
	}
	return map[string]interface{}{"LoadState": "loaded"}, nil
}

func (m *MockDiagnosticsConnection) GetUnitProperty(_ context.Context, _, _ string) (*dbusapi.Property, error) {
	return nil, nil
}

func (m *MockDiagnosticsConnection) ResetFailedUnit(_ context.Context, _ string) error {
	return nil
}

func (m *MockDiagnosticsConnection) StartUnit(_ context.Context, _, _ string) (chan string, error) {
	return nil, nil
}

func (m *MockDiagnosticsConnection) StopUnit(_ context.Context, _, _ string) (chan string, error) {
	return nil, nil
}

func (m *MockDiagnosticsConnection) RestartUnit(_ context.Context, _, _ string) (chan string, error) {
	return nil, nil
}

func (m *MockDiagnosticsConnection) Reload(_ context.Context) error {
	return nil
}

func (m *MockDiagnosticsConnection) Close() error {
	return nil
}

func TestCheckGeneratorBinaryExists_Found(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{
			"/usr/lib/systemd/system-generators/podman-system-generator": true,
		},
	}

	exists, err := CheckGeneratorBinaryExists("/usr/lib/systemd/system-generators/podman-system-generator", fs, logger)

	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckGeneratorBinaryExists_NotFound(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{},
	}

	exists, err := CheckGeneratorBinaryExists("/usr/lib/systemd/system-generators/podman-system-generator", fs, logger)

	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckGeneratorBinaryExists_ErrorAccessing(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		fileError: errors.New("permission denied"),
	}

	exists, err := CheckGeneratorBinaryExists("/usr/lib/systemd/system-generators/podman-system-generator", fs, logger)

	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestCheckUnitLoaded_Loaded(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{
			"test.service": true,
		},
	}

	loaded, err := CheckUnitLoaded(context.Background(), "test.service", factory, false, logger)

	assert.NoError(t, err)
	assert.True(t, loaded)
}

func TestCheckUnitLoaded_NotLoaded(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{},
	}

	loaded, err := CheckUnitLoaded(context.Background(), "test.service", factory, false, logger)

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestCheckUnitLoaded_ConnectionError(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	factory := &MockDiagnosticsConnectionFactory{
		connError: errors.New("dbus connection failed"),
	}

	loaded, err := CheckUnitLoaded(context.Background(), "test.service", factory, false, logger)

	assert.Error(t, err)
	assert.False(t, loaded)
	assert.Contains(t, err.Error(), "dbus connection failed")
}

func TestDiagnoseGeneratorIssues_AllHealthy(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{
			"/usr/lib/systemd/system-generators/podman-system-generator": true,
			"/etc/containers/systemd/test.container":                     true,
		},
	}
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{
			"test.service": true,
		},
	}

	artifacts := []string{"/etc/containers/systemd/test.container"}
	issues := DiagnoseGeneratorIssues(
		context.Background(),
		"/usr/lib/systemd/system-generators/podman-system-generator",
		artifacts,
		fs,
		factory,
		false,
		logger,
	)

	assert.Empty(t, issues)
}

func TestDiagnoseGeneratorIssues_GeneratorMissing(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{
			"/etc/containers/systemd/test.container": true,
		},
	}
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{},
	}

	artifacts := []string{"/etc/containers/systemd/test.container"}
	issues := DiagnoseGeneratorIssues(
		context.Background(),
		"/usr/lib/systemd/system-generators/podman-system-generator",
		artifacts,
		fs,
		factory,
		false,
		logger,
	)

	assert.Len(t, issues, 1)
	assert.Equal(t, "generator_missing", issues[0].Type)
	assert.Contains(t, issues[0].Message, "Quadlet generator binary not found")
	assert.NotEmpty(t, issues[0].Suggestions)
}

func TestDiagnoseGeneratorIssues_UnitNotGenerated(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{
			"/usr/lib/systemd/system-generators/podman-system-generator": true,
			"/etc/containers/systemd/test.container":                     true,
		},
	}
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{},
	}

	artifacts := []string{"/etc/containers/systemd/test.container"}
	issues := DiagnoseGeneratorIssues(
		context.Background(),
		"/usr/lib/systemd/system-generators/podman-system-generator",
		artifacts,
		fs,
		factory,
		false,
		logger,
	)

	assert.Len(t, issues, 1)
	assert.Equal(t, "unit_not_generated", issues[0].Type)
	assert.Contains(t, issues[0].Message, "test.container exists but test.service not loaded")
	assert.NotEmpty(t, issues[0].Suggestions)
}

func TestDiagnoseGeneratorIssues_MultipleArtifactsPartialFailure(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	fs := &MockFileSystem{
		existingFiles: map[string]bool{
			"/usr/lib/systemd/system-generators/podman-system-generator": true,
			"/etc/containers/systemd/test1.container":                    true,
			"/etc/containers/systemd/test2.container":                    true,
		},
	}
	factory := &MockDiagnosticsConnectionFactory{
		loadedUnits: map[string]bool{
			"test1.service": true,
			// test2.service not loaded
		},
	}

	artifacts := []string{
		"/etc/containers/systemd/test1.container",
		"/etc/containers/systemd/test2.container",
	}
	issues := DiagnoseGeneratorIssues(
		context.Background(),
		"/usr/lib/systemd/system-generators/podman-system-generator",
		artifacts,
		fs,
		factory,
		false,
		logger,
	)

	assert.Len(t, issues, 1)
	assert.Equal(t, "unit_not_generated", issues[0].Type)
	assert.Contains(t, issues[0].Message, "test2.container")
}

func TestFormatDiagnosticIssue_GeneratorMissing(t *testing.T) {
	issue := DiagnosticIssue{
		Type:    "generator_missing",
		Message: "Generator binary not found at /path/to/generator",
		Suggestions: []string{
			"Install podman-system-generator",
			"Check podman version",
		},
	}

	output := FormatDiagnosticIssue(issue)

	assert.Contains(t, output, "Generator binary not found")
	assert.Contains(t, output, "Install podman-system-generator")
	assert.Contains(t, output, "Check podman version")
}

func TestFormatDiagnosticIssue_UnitNotGenerated(t *testing.T) {
	issue := DiagnosticIssue{
		Type:    "unit_not_generated",
		Message: "test.container exists but test.service not loaded",
		Suggestions: []string{
			"Run: systemctl daemon-reload",
			"Check generator logs",
		},
	}

	output := FormatDiagnosticIssue(issue)

	assert.Contains(t, output, "test.container exists but test.service not loaded")
	assert.Contains(t, output, "systemctl daemon-reload")
	assert.Contains(t, output, "Check generator logs")
}

func TestArtifactPathToUnitName_Container(t *testing.T) {
	unitName := ArtifactPathToUnitName("/etc/containers/systemd/test.container")
	assert.Equal(t, "test.service", unitName)
}

func TestArtifactPathToUnitName_Network(t *testing.T) {
	unitName := ArtifactPathToUnitName("/etc/containers/systemd/mynet.network")
	assert.Equal(t, "mynet-network.service", unitName)
}

func TestArtifactPathToUnitName_Volume(t *testing.T) {
	unitName := ArtifactPathToUnitName("/etc/containers/systemd/myvol.volume")
	assert.Equal(t, "myvol-volume.service", unitName)
}

func TestArtifactPathToUnitName_Build(t *testing.T) {
	unitName := ArtifactPathToUnitName("/etc/containers/systemd/mybuild.build")
	assert.Equal(t, "mybuild.service", unitName)
}

func TestArtifactPathToUnitName_Image(t *testing.T) {
	unitName := ArtifactPathToUnitName("/etc/containers/systemd/myimage.image")
	assert.Equal(t, "myimage.service", unitName)
}
