package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestContainerConfigYAMLMarshaling(t *testing.T) {
	// Create a sample config
	config := Container{
		Image:           "nginx:latest",
		Label:           []string{"com.example.key1=value1", "com.example.key2=value2"},
		PublishPort:     []string{"8080:80", "443:443"},
		Environment:     map[string]string{"ENV1": "value1", "ENV2": "value2"},
		EnvironmentFile: "/path/to/env/file",
		Volume:          []string{"/host/path:/container/path", "named-volume:/data"},
		Network:         []string{"host", "container-network"},
		Exec:            []string{"nginx", "-g", "daemon off;"},
		Entrypoint:      []string{"/docker-entrypoint.sh"},
		User:            "nginx",
		Group:           "nginx",
		WorkingDir:      "/usr/share/nginx/html",
		PodmanArgs:      []string{"--log-level=debug", "--cgroup-manager=systemd"},
		RunInit:         true,
		Notify:          true,
		Privileged:      false,
		ReadOnly:        true,
		SecurityLabel:   []string{"label=value", "level=s0"},
		HostName:        "web-server",
		Secrets: []Secret{
			{
				Name:   "web-cert",
				Type:   "mount",
				Target: "/etc/certs",
				UID:    1000,
				GID:    1000,
				Mode:   "0400",
			},
			{
				Name:   "api-key",
				Type:   "env",
				Target: "API_KEY",
			},
		},
	}

	// Test marshaling to YAML
	yamlData, err := yaml.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Test unmarshaling from YAML
	var unmarshaled Container
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	assert.NoError(t, err)

	// Verify the unmarshaled data matches the original
	assert.Equal(t, config.Image, unmarshaled.Image)
	assert.ElementsMatch(t, config.Label, unmarshaled.Label)
	assert.ElementsMatch(t, config.PublishPort, unmarshaled.PublishPort)
	assert.Equal(t, config.Environment, unmarshaled.Environment)
	assert.Equal(t, config.EnvironmentFile, unmarshaled.EnvironmentFile)
	assert.ElementsMatch(t, config.Volume, unmarshaled.Volume)
	assert.ElementsMatch(t, config.Network, unmarshaled.Network)
	assert.ElementsMatch(t, config.Exec, unmarshaled.Exec)
	assert.ElementsMatch(t, config.Entrypoint, unmarshaled.Entrypoint)
	assert.Equal(t, config.User, unmarshaled.User)
	assert.Equal(t, config.Group, unmarshaled.Group)
	assert.Equal(t, config.WorkingDir, unmarshaled.WorkingDir)
	assert.ElementsMatch(t, config.PodmanArgs, unmarshaled.PodmanArgs)
	assert.Equal(t, config.RunInit, unmarshaled.RunInit)
	assert.Equal(t, config.Notify, unmarshaled.Notify)
	assert.Equal(t, config.Privileged, unmarshaled.Privileged)
	assert.Equal(t, config.ReadOnly, unmarshaled.ReadOnly)
	assert.ElementsMatch(t, config.SecurityLabel, unmarshaled.SecurityLabel)
	assert.Equal(t, config.HostName, unmarshaled.HostName)
	assert.Len(t, unmarshaled.Secrets, len(config.Secrets))

	// Test specific secret config fields
	for i, secret := range config.Secrets {
		assert.Equal(t, secret.Name, unmarshaled.Secrets[i].Name)
		assert.Equal(t, secret.Type, unmarshaled.Secrets[i].Type)
		assert.Equal(t, secret.Target, unmarshaled.Secrets[i].Target)
		assert.Equal(t, secret.UID, unmarshaled.Secrets[i].UID)
		assert.Equal(t, secret.GID, unmarshaled.Secrets[i].GID)
		assert.Equal(t, secret.Mode, unmarshaled.Secrets[i].Mode)
	}
}

func TestContainerConfigYAMLUnmarshaling(t *testing.T) {
	yamlData := `
image: postgres:13
label:
  - app=database
  - environment=production
publish:
  - 5432:5432
environment:
  POSTGRES_USER: admin
  POSTGRES_PASSWORD: secret
  POSTGRES_DB: myapp
environment_file: /etc/secrets/db-env
volume:
  - pgdata:/var/lib/postgresql/data
network:
  - backend
command:
  - postgres
user: postgres
group: postgres
working_dir: /var/lib/postgresql
run_init: false
notify: true
privileged: false
read_only: false
hostname: db-server
secrets:
  - name: db-cert
    type: mount
    target: /etc/certs
    uid: 999
    gid: 999
    mode: "0400"
  - name: db-password
    type: env
    target: DB_PASSWORD
`

	var config Container
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)

	// Verify fields were properly unmarshaled
	assert.Equal(t, "postgres:13", config.Image)
	assert.ElementsMatch(t, []string{"app=database", "environment=production"}, config.Label)
	assert.ElementsMatch(t, []string{"5432:5432"}, config.PublishPort)
	assert.Equal(t, map[string]string{
		"POSTGRES_USER":     "admin",
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_DB":       "myapp",
	}, config.Environment)
	assert.Equal(t, "/etc/secrets/db-env", config.EnvironmentFile)
	assert.ElementsMatch(t, []string{"pgdata:/var/lib/postgresql/data"}, config.Volume)
	assert.ElementsMatch(t, []string{"backend"}, config.Network)
	assert.ElementsMatch(t, []string{"postgres"}, config.Exec)
	assert.Equal(t, "postgres", config.User)
	assert.Equal(t, "postgres", config.Group)
	assert.Equal(t, "/var/lib/postgresql", config.WorkingDir)
	assert.False(t, config.RunInit)
	assert.True(t, config.Notify)
	assert.False(t, config.Privileged)
	assert.False(t, config.ReadOnly)
	assert.Equal(t, "db-server", config.HostName)

	// Verify secrets
	assert.Len(t, config.Secrets, 2)
	assert.Equal(t, "db-cert", config.Secrets[0].Name)
	assert.Equal(t, "mount", config.Secrets[0].Type)
	assert.Equal(t, "/etc/certs", config.Secrets[0].Target)
	assert.Equal(t, 999, config.Secrets[0].UID)
	assert.Equal(t, 999, config.Secrets[0].GID)
	assert.Equal(t, "0400", config.Secrets[0].Mode)

	assert.Equal(t, "db-password", config.Secrets[1].Name)
	assert.Equal(t, "env", config.Secrets[1].Type)
	assert.Equal(t, "DB_PASSWORD", config.Secrets[1].Target)
}

func TestSecretConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		secret  Secret
		isValid bool
	}{
		{
			name: "Valid mount secret",
			secret: Secret{
				Name:   "cert",
				Type:   "mount",
				Target: "/etc/ssl/certs",
				UID:    1000,
				GID:    1000,
				Mode:   "0400",
			},
			isValid: true,
		},
		{
			name: "Valid env secret",
			secret: Secret{
				Name:   "api-key",
				Type:   "env",
				Target: "API_KEY",
			},
			isValid: true,
		},
		{
			name: "Invalid secret type",
			secret: Secret{
				Name:   "invalid",
				Type:   "unknown",
				Target: "somewhere",
			},
			isValid: false,
		},
		{
			name: "Missing name",
			secret: Secret{
				Type:   "mount",
				Target: "/etc/certs",
			},
			isValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.secret.Validate()

			if tc.isValid {
				assert.NoError(t, err, "Expected valid secret config")
			} else {
				assert.Error(t, err, "Expected invalid secret config")
			}
		})
	}
}
