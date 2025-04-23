package unit

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestServicesDependencyTree(t *testing.T) {
	// Create a mock project with a simple dependency tree
	// db <- webapp <- proxy
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
			"proxy": types.ServiceConfig{
				Name:  "proxy",
				Image: "nginx:latest",
				DependsOn: types.DependsOnConfig{
					"webapp": types.ServiceDependency{},
				},
			},
		},
	}

	// Build the dependency tree
	deps := BuildServiceDependencyTree(project)

	// Check that the dependency tree is correct
	assert.Len(t, deps, 3)

	// Check db has no dependencies
	assert.Empty(t, deps["db"].Dependencies)
	assert.Len(t, deps["db"].DependentServices, 1)
	assert.Contains(t, deps["db"].DependentServices, "webapp")

	// Check webapp depends on db
	assert.Len(t, deps["webapp"].Dependencies, 1)
	assert.Contains(t, deps["webapp"].Dependencies, "db")
	assert.Len(t, deps["webapp"].DependentServices, 1)
	assert.Contains(t, deps["webapp"].DependentServices, "proxy")

	// Check proxy depends on webapp
	assert.Len(t, deps["proxy"].Dependencies, 1)
	assert.Contains(t, deps["proxy"].Dependencies, "webapp")
	assert.Empty(t, deps["proxy"].DependentServices)
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

	// Build the dependency tree
	deps := BuildServiceDependencyTree(project)

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

	// Apply dependencies
	ApplyDependencyRelationships(&dbUnit, "db", deps, project.Name)

	// Check that db has dependencies on networks and volumes, and has PartOf for webapp
	// Should have 2 dependencies - 1 network (backend) and 1 volume (db-data)
	assert.Len(t, dbUnit.Systemd.After, 2)
	assert.Contains(t, dbUnit.Systemd.After, "test-project-backend-network.service")
	assert.Contains(t, dbUnit.Systemd.After, "test-project-db-data-volume.service")
	assert.Len(t, dbUnit.Systemd.Requires, 2)
	assert.Contains(t, dbUnit.Systemd.Requires, "test-project-backend-network.service")
	assert.Contains(t, dbUnit.Systemd.Requires, "test-project-db-data-volume.service")
	assert.Len(t, dbUnit.Systemd.PartOf, 1)
	assert.Contains(t, dbUnit.Systemd.PartOf, "test-project-webapp.service")

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
	ApplyDependencyRelationships(&webappUnit, "webapp", deps, project.Name)

	// Check that webapp has After/Requires for db, networks, volumes and PartOf for proxy
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
	assert.Len(t, webappUnit.Systemd.PartOf, 1)
	assert.Contains(t, webappUnit.Systemd.PartOf, "test-project-proxy.service")

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
	ApplyDependencyRelationships(&proxyUnit, "proxy", deps, project.Name)

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
