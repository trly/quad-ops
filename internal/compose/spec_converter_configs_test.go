package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

func TestSpecConverter_ConvertConfigMounts(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "app.conf")
	require.NoError(t, os.WriteFile(configFile, []byte("config content"), 0644))

	tests := []struct {
		name          string
		configs       []types.ServiceConfigObjConfig
		projectName   string
		projectConfig map[string]types.ConfigObjConfig
		serviceName   string
		envVars       map[string]string
		want          int
		wantErr       bool
		checkMount    func(t *testing.T, mounts []service.Mount)
	}{
		{
			name:    "no configs",
			configs: []types.ServiceConfigObjConfig{},
			want:    0,
		},
		{
			name: "file source config",
			configs: []types.ServiceConfigObjConfig{
				{Source: "app-config", Target: "/etc/app.conf"},
			},
			projectName: "test-project",
			projectConfig: map[string]types.ConfigObjConfig{
				"app-config": {
					Name: "app-config",
					File: configFile,
				},
			},
			serviceName: "web",
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, configFile, mounts[0].Source)
				assert.Equal(t, "/etc/app.conf", mounts[0].Target)
				assert.True(t, mounts[0].ReadOnly)
			},
		},
		{
			name: "content source config",
			configs: []types.ServiceConfigObjConfig{
				{Source: "inline-config", Target: "/app/config.txt"},
			},
			projectName: "test-project",
			projectConfig: map[string]types.ConfigObjConfig{
				"inline-config": {
					Name:    "inline-config",
					Content: "inline config data",
				},
			},
			serviceName: "web",
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, "/app/config.txt", mounts[0].Target)
				assert.True(t, mounts[0].ReadOnly)
				content, err := os.ReadFile(mounts[0].Source)
				require.NoError(t, err)
				assert.Equal(t, "inline config data", string(content))
			},
		},
		{
			name: "environment source config",
			configs: []types.ServiceConfigObjConfig{
				{Source: "env-config", Target: "/run/config"},
			},
			projectName: "test-project",
			projectConfig: map[string]types.ConfigObjConfig{
				"env-config": {
					Name:        "env-config",
					Environment: "TEST_CONFIG_VAR",
				},
			},
			serviceName: "web",
			envVars:     map[string]string{"TEST_CONFIG_VAR": "env config value"},
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, "/run/config", mounts[0].Target)
				assert.True(t, mounts[0].ReadOnly)
				content, err := os.ReadFile(mounts[0].Source)
				require.NoError(t, err)
				assert.Equal(t, "env config value", string(content))
			},
		},
		{
			name: "external config skipped",
			configs: []types.ServiceConfigObjConfig{
				{Source: "external-config", Target: "/etc/external.conf"},
			},
			projectName: "test-project",
			projectConfig: map[string]types.ConfigObjConfig{
				"external-config": {
					Name:     "external-config",
					External: types.External(true),
				},
			},
			serviceName: "web",
			want:        0,
		},
		{
			name: "config not found error",
			configs: []types.ServiceConfigObjConfig{
				{Source: "missing-config", Target: "/etc/missing.conf"},
			},
			projectName:   "test-project",
			projectConfig: map[string]types.ConfigObjConfig{},
			serviceName:   "web",
			wantErr:       true,
		},
		{
			name: "default target path",
			configs: []types.ServiceConfigObjConfig{
				{Source: "default-config"},
			},
			projectName: "test-project",
			projectConfig: map[string]types.ConfigObjConfig{
				"default-config": {
					Name:    "default-config",
					Content: "default",
				},
			},
			serviceName: "web",
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, "/default-config", mounts[0].Target)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			converter := NewSpecConverter(tmpDir)
			project := &types.Project{
				Name:    tt.projectName,
				Configs: tt.projectConfig,
			}

			got, err := converter.convertConfigMounts(tt.configs, project, tt.serviceName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.want)

			if tt.checkMount != nil && len(got) > 0 {
				tt.checkMount(t, got)
			}
		})
	}
}

func TestSpecConverter_ConvertSecretMounts(t *testing.T) {
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	require.NoError(t, os.WriteFile(secretFile, []byte("secret content"), 0600))

	tests := []struct {
		name          string
		secrets       []types.ServiceSecretConfig
		projectName   string
		projectSecret map[string]types.SecretConfig
		serviceName   string
		envVars       map[string]string
		want          int
		wantErr       bool
		checkMount    func(t *testing.T, mounts []service.Mount)
	}{
		{
			name:    "no secrets",
			secrets: []types.ServiceSecretConfig{},
			want:    0,
		},
		{
			name: "file source secret",
			secrets: []types.ServiceSecretConfig{
				{Source: "db-password", Target: "/run/secrets/db-pass"},
			},
			projectName: "test-project",
			projectSecret: map[string]types.SecretConfig{
				"db-password": {
					Name: "db-password",
					File: secretFile,
				},
			},
			serviceName: "web",
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, secretFile, mounts[0].Source)
				assert.Equal(t, "/run/secrets/db-pass", mounts[0].Target)
				assert.True(t, mounts[0].ReadOnly)
			},
		},
		{
			name: "content source secret",
			secrets: []types.ServiceSecretConfig{
				{Source: "api-key"},
			},
			projectName: "test-project",
			projectSecret: map[string]types.SecretConfig{
				"api-key": {
					Name:    "api-key",
					Content: "supersecret123",
				},
			},
			serviceName: "web",
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, "/run/secrets/api-key", mounts[0].Target)
				assert.True(t, mounts[0].ReadOnly)
				content, err := os.ReadFile(mounts[0].Source)
				require.NoError(t, err)
				assert.Equal(t, "supersecret123", string(content))
			},
		},
		{
			name: "environment source secret",
			secrets: []types.ServiceSecretConfig{
				{Source: "token"},
			},
			projectName: "test-project",
			projectSecret: map[string]types.SecretConfig{
				"token": {
					Name:        "token",
					Environment: "SECRET_TOKEN",
				},
			},
			serviceName: "web",
			envVars:     map[string]string{"SECRET_TOKEN": "mytoken123"},
			want:        1,
			checkMount: func(t *testing.T, mounts []service.Mount) {
				assert.Equal(t, service.MountTypeBind, mounts[0].Type)
				assert.Equal(t, "/run/secrets/token", mounts[0].Target)
				content, err := os.ReadFile(mounts[0].Source)
				require.NoError(t, err)
				assert.Equal(t, "mytoken123", string(content))
			},
		},
		{
			name: "external secret skipped",
			secrets: []types.ServiceSecretConfig{
				{Source: "external-secret"},
			},
			projectName: "test-project",
			projectSecret: map[string]types.SecretConfig{
				"external-secret": {
					Name:     "external-secret",
					External: types.External(true),
				},
			},
			serviceName: "web",
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			converter := NewSpecConverter(tmpDir)
			project := &types.Project{
				Name:    tt.projectName,
				Secrets: tt.projectSecret,
			}

			got, err := converter.convertSecretMounts(tt.secrets, project, tt.serviceName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.want)

			if tt.checkMount != nil && len(got) > 0 {
				tt.checkMount(t, got)
			}
		})
	}
}

func TestSpecConverter_ValidateProject(t *testing.T) {
	tests := []struct {
		name    string
		project *types.Project
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid project",
			project: &types.Project{
				Name: "test",
				Configs: map[string]types.ConfigObjConfig{
					"valid": {
						Name:    "valid",
						Content: "data",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config with driver rejected",
			project: &types.Project{
				Name: "test",
				Configs: map[string]types.ConfigObjConfig{
					"swarm-config": {
						Name:   "swarm-config",
						Driver: "swarm-driver",
					},
				},
			},
			wantErr: true,
			errMsg:  "Swarm-specific",
		},
		{
			name: "secret with driver rejected",
			project: &types.Project{
				Name: "test",
				Secrets: map[string]types.SecretConfig{
					"swarm-secret": {
						Name:   "swarm-secret",
						Driver: "swarm-driver",
					},
				},
			},
			wantErr: true,
			errMsg:  "Swarm-specific",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter("/tmp")
			err := converter.validateProject(tt.project)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
