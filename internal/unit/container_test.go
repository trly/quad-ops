package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestContainer_GetSystemdUnit(t *testing.T) {
	container := Container{
		Name:     "test-container",
		UnitType: "container",
	}

	assert.Equal(t, "test-container", container.GetUnitName())
	assert.Equal(t, "container", container.GetUnitType())
	assert.Equal(t, "test-container.service", container.GetServiceName())
}

func TestFromComposeService(t *testing.T) {
	// Setup test data
	serviceName := "test-service"
	image := "nginx:latest"
	var labels = types.Labels{}
	labels.Add("label1", "value1")
	labels.Add("label2", "value2")

	envFiles := []types.EnvFile{
		{
			Path:     "./env.local",
			Format:   "env",
			Required: true,
		},
	}

	// Create a port config
	publishedPort := "8080"
	targetPort := uint32(80)
	portConfig := []types.ServicePortConfig{
		{
			Published: publishedPort,
			Target:    targetPort,
		},
	}

	// Environment variables
	envValue1 := "env_value1"
	envValue2 := "env_value2"
	env := map[string]*string{
		"ENV_VAR1": &envValue1,
		"ENV_VAR2": &envValue2,
	}

	// Volume configurations
	volumes := []types.ServiceVolumeConfig{
		{
			Source: "./data",
			Target: "/app/data",
		},
		{
			Source: "logs",
			Target: "/var/logs",
		},
	}

	// Network configurations
	networks := map[string]*types.ServiceNetworkConfig{
		"frontend": {
			Aliases: []string{"web"},
		},
		"backend": {
			Aliases: []string{"api"},
		},
		// Add a network without aliases to test the nil case
		"default": {
			// No aliases
		},
	}

	// Command and entrypoint
	command := []string{"nginx", "-g", "daemon off;"}
	entrypoint := []string{"/docker-entrypoint.sh"}

	// Other properties
	user := "nginx"
	workingDir := "/app"
	init := true
	privileged := true
	readOnly := true
	securityOpt := []string{"label=user:USER", "label=role:ROLE"}
	hostname := "web-server"

	// Secret configurations
	uid := "1000"
	gid := "1000"
	secrets := []types.ServiceSecretConfig{
		{
			Source: "db_password",
			Target: "app_password",
			UID:    uid,
			GID:    gid,
		},
	}

	// Create the service config
	service := types.ServiceConfig{
		Name:        serviceName,
		Labels:      labels,
		Image:       image,
		Ports:       portConfig,
		Environment: env,
		EnvFiles:    envFiles,
		Volumes:     volumes,
		Networks:    networks,
		Command:     command,
		Entrypoint:  entrypoint,
		User:        user,
		WorkingDir:  workingDir,
		Init:        &init,
		Privileged:  privileged,
		ReadOnly:    readOnly,
		SecurityOpt: securityOpt,
		Hostname:    hostname,
		Secrets:     secrets,
	}

	container := NewContainer(serviceName)
	// Call the function being tested
	container = container.FromComposeService(service)

	// Assert all properties were transferred correctly
	assert.Equal(t, serviceName, container.Name)
	assert.Equal(t, "container", container.UnitType)
	assert.Equal(t, image, container.Image)
	assert.ElementsMatch(t, labels.AsList(), container.Label)

	// Verify ports
	expectedPort := "8080:80"
	assert.Contains(t, container.PublishPort, expectedPort)

	// Verify environment variables
	assert.Equal(t, envValue1, container.Environment["ENV_VAR1"])
	assert.Equal(t, envValue2, container.Environment["ENV_VAR2"])

	// Verify env files
	assert.ElementsMatch(t, []string{"./env.local"}, container.EnvironmentFile)

	// Verify volumes
	assert.Contains(t, container.Volume, "./data:/app/data")
	assert.Contains(t, container.Volume, "logs:/var/logs")

	// Verify networks
	assert.ElementsMatch(t, []string{"web", "api", "default"}, container.Network)

	// Verify command and entrypoint
	assert.Equal(t, command, container.Exec)
	assert.Equal(t, entrypoint, container.Entrypoint)

	// Verify other properties
	assert.Equal(t, user, container.User)
	assert.Equal(t, workingDir, container.WorkingDir)
	assert.Equal(t, init, *container.RunInit)
	assert.Equal(t, privileged, container.Privileged)
	assert.Equal(t, readOnly, container.ReadOnly)
	assert.ElementsMatch(t, securityOpt, container.SecurityLabel)
	assert.Equal(t, hostname, container.HostName)

	// Verify secrets
	assert.Equal(t, 1, len(container.Secrets))
	assert.Equal(t, "db_password", container.Secrets[0].Source)
	assert.Equal(t, "app_password", container.Secrets[0].Target)
	assert.Equal(t, uid, container.Secrets[0].UID)
	assert.Equal(t, gid, container.Secrets[0].GID)
	assert.Equal(t, "0644", container.Secrets[0].Mode)
}
