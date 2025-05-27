package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/dependency"
)

func TestDependencyGraphApplyRelationships(t *testing.T) {
	// Create a mock project with dependencies
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"db": types.ServiceConfig{
				Name:  "db",
				Image: "mariadb:latest",
			},
			"webapp": types.ServiceConfig{
				Name:  "webapp",
				Image: "wordpress:latest",
				DependsOn: types.DependsOnConfig{
					"db": types.ServiceDependency{},
				},
			},
		},
	}

	// Build the dependency graph
	graph, err := dependency.BuildServiceDependencyGraph(project)
	require.NoError(t, err)

	// Create container for webapp
	unit := QuadletUnit{
		Name:      "test-project-webapp",
		Type:      "container",
		Container: Container{},
		Systemd:   SystemdConfig{},
	}

	// Apply dependencies
	err = ApplyDependencyRelationships(&unit, "webapp", graph, "test-project")
	require.NoError(t, err)

	// Check that webapp has After/Requires for db
	assert.Len(t, unit.Systemd.After, 1)
	assert.Contains(t, unit.Systemd.After, "test-project-db.service")
	assert.Len(t, unit.Systemd.Requires, 1)
	assert.Contains(t, unit.Systemd.Requires, "test-project-db.service")
	assert.Empty(t, unit.Systemd.PartOf)
}

func TestDependencyPartOfRelationships(t *testing.T) {
	// Create a mock project with a simple dependency tree plus networks and volumes
	// db <- webapp <- proxy
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"db": types.ServiceConfig{
				Name:  "db",
				Image: "mariadb:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"backend": {},
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "db-data",
						Target: "/var/lib/mysql",
					},
				},
			},
			"webapp": types.ServiceConfig{
				Name:  "webapp",
				Image: "wordpress:latest",
				DependsOn: types.DependsOnConfig{
					"db": types.ServiceDependency{},
				},
				Networks: map[string]*types.ServiceNetworkConfig{
					"backend":  {},
					"frontend": {},
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "wp-content",
						Target: "/var/www/html/wp-content",
					},
				},
			},
			"proxy": types.ServiceConfig{
				Name:  "proxy",
				Image: "nginx:latest",
				DependsOn: types.DependsOnConfig{
					"webapp": types.ServiceDependency{},
				},
				Networks: map[string]*types.ServiceNetworkConfig{
					"frontend": {},
				},
			},
		},
		Networks: map[string]types.NetworkConfig{
			"backend": {
				Name: "backend",
			},
			"frontend": {
				Name: "frontend",
			},
		},
		Volumes: map[string]types.VolumeConfig{
			"db-data": {
				Name: "db-data",
			},
			"wp-content": {
				Name: "wp-content",
			},
		},
	}

	// Create container for db
	prefixedName := "test-project-db"
	dbContainer := NewContainer(prefixedName)
	dbContainer = dbContainer.FromComposeService(project.Services["db"], project.Name)
	dbUnit := QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *dbContainer,
		Systemd:   SystemdConfig{},
	}

	// Build graph from project for testing
	graph, err := dependency.BuildServiceDependencyGraph(project)
	require.NoError(t, err)

	// Apply dependencies
	err = ApplyDependencyRelationships(&dbUnit, "db", graph, project.Name)
	assert.NoError(t, err)

	// Check that db has dependencies on networks and volumes, but no PartOf to avoid circular dependencies
	// Should have 2 dependencies - 1 network (backend) and 1 volume (db-data)
	assert.Len(t, dbUnit.Systemd.After, 2)
	assert.Contains(t, dbUnit.Systemd.After, "test-project-backend-network.service")
	assert.Contains(t, dbUnit.Systemd.After, "test-project-db-data-volume.service")
	assert.Len(t, dbUnit.Systemd.Requires, 2)
	assert.Contains(t, dbUnit.Systemd.Requires, "test-project-backend-network.service")
	assert.Contains(t, dbUnit.Systemd.Requires, "test-project-db-data-volume.service")
	assert.Empty(t, dbUnit.Systemd.PartOf)

	// Create container for webapp
	prefixedName = "test-project-webapp"
	webappContainer := NewContainer(prefixedName)
	webappContainer = webappContainer.FromComposeService(project.Services["webapp"], project.Name)
	webappUnit := QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *webappContainer,
		Systemd:   SystemdConfig{},
	}

	// Apply dependencies
	err = ApplyDependencyRelationships(&webappUnit, "webapp", graph, project.Name)
	assert.NoError(t, err)

	// Check that webapp has After/Requires for db, networks, volumes but no PartOf to avoid circular dependencies
	// Should have 4 dependencies - db service, 2 networks (backend, frontend), and 1 volume (wp-content)
	assert.Len(t, webappUnit.Systemd.After, 4)
	assert.Contains(t, webappUnit.Systemd.After, "test-project-db.service")
	assert.Contains(t, webappUnit.Systemd.After, "test-project-backend-network.service")
	assert.Contains(t, webappUnit.Systemd.After, "test-project-frontend-network.service")
	assert.Contains(t, webappUnit.Systemd.After, "test-project-wp-content-volume.service")
	assert.Len(t, webappUnit.Systemd.Requires, 4)
	assert.Contains(t, webappUnit.Systemd.Requires, "test-project-db.service")
	assert.Contains(t, webappUnit.Systemd.Requires, "test-project-backend-network.service")
	assert.Contains(t, webappUnit.Systemd.Requires, "test-project-frontend-network.service")
	assert.Contains(t, webappUnit.Systemd.Requires, "test-project-wp-content-volume.service")
	assert.Empty(t, webappUnit.Systemd.PartOf)

	// Create container for proxy
	prefixedName = "test-project-proxy"
	proxyContainer := NewContainer(prefixedName)
	proxyContainer = proxyContainer.FromComposeService(project.Services["proxy"], project.Name)
	proxyUnit := QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *proxyContainer,
		Systemd:   SystemdConfig{},
	}

	// Apply dependencies
	err = ApplyDependencyRelationships(&proxyUnit, "proxy", graph, project.Name)
	assert.NoError(t, err)

	// Check that proxy has After/Requires for webapp and network but no PartOf
	// Should have 2 dependencies - webapp service and frontend network
	assert.Len(t, proxyUnit.Systemd.After, 2)
	assert.Contains(t, proxyUnit.Systemd.After, "test-project-webapp.service")
	assert.Contains(t, proxyUnit.Systemd.After, "test-project-frontend-network.service")
	assert.Len(t, proxyUnit.Systemd.Requires, 2)
	assert.Contains(t, proxyUnit.Systemd.Requires, "test-project-webapp.service")
	assert.Contains(t, proxyUnit.Systemd.Requires, "test-project-frontend-network.service")
	assert.Empty(t, proxyUnit.Systemd.PartOf)
}
