package compose

import (
	"fmt"
	"os"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, types.ServicePortConfig(types.ServicePortConfig{Mode: "ingress", Published: "443", Target: 8043, Protocol: "tcp"}), project.Services["frontend"].Ports[0])
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
