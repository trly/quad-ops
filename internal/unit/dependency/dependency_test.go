package dependency

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/unit/model"
)

func TestServicesDependencyTree(t *testing.T) {
	// Create a project with services that have dependencies
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"db": types.ServiceConfig{
				Name: "db",
			},
			"redis": types.ServiceConfig{
				Name: "redis",
			},
			"app": types.ServiceConfig{
				Name: "app",
				DependsOn: types.DependsOnConfig{
					"db":    types.ServiceDependency{},
					"redis": types.ServiceDependency{},
				},
			},
			"web": types.ServiceConfig{
				Name: "web",
				DependsOn: types.DependsOnConfig{
					"app": types.ServiceDependency{},
				},
			},
		},
	}

	// Build the dependency tree
	dependencyTree := BuildServiceDependencyTree(project)

	// Verify app depends on db and redis
	assert.Contains(t, dependencyTree["app"].Dependencies, "db")
	// Verify web depends on app
	assert.Contains(t, dependencyTree["web"].Dependencies, "app")
	// Verify app has web as dependent
	assert.Contains(t, dependencyTree["app"].DependentServices, "web")
	// Verify db has app as dependent
	assert.Contains(t, dependencyTree["db"].DependentServices, "app")
}

func TestDependencyPartOfRelationships(t *testing.T) {
	// Create a project with services that have dependencies
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"db": types.ServiceConfig{
				Name: "db",
			},
			"redis": types.ServiceConfig{
				Name: "redis",
			},
			"app": types.ServiceConfig{
				Name: "app",
				DependsOn: types.DependsOnConfig{
					"db":    types.ServiceDependency{},
					"redis": types.ServiceDependency{},
				},
			},
			"web": types.ServiceConfig{
				Name: "web",
				DependsOn: types.DependsOnConfig{
					"app": types.ServiceDependency{},
				},
			},
		},
	}

	// Build the dependency tree
	dependencyTree := BuildServiceDependencyTree(project)

	// Create quadlet units for each service
	dbUnit := &model.QuadletUnitConfig{
		Name:    "test-db",
		Type:    "container",
		Systemd: model.SystemdConfig{},
	}

	redisUnit := &model.QuadletUnitConfig{
		Name:    "test-redis",
		Type:    "container",
		Systemd: model.SystemdConfig{},
	}

	appUnit := &model.QuadletUnitConfig{
		Name:    "test-app",
		Type:    "container",
		Systemd: model.SystemdConfig{},
	}

	webUnit := &model.QuadletUnitConfig{
		Name:    "test-web",
		Type:    "container",
		Systemd: model.SystemdConfig{},
	}

	// Apply dependencies
	ApplyDependencyRelationships(dbUnit, "db", dependencyTree, "test")
	ApplyDependencyRelationships(redisUnit, "redis", dependencyTree, "test")
	ApplyDependencyRelationships(appUnit, "app", dependencyTree, "test")
	ApplyDependencyRelationships(webUnit, "web", dependencyTree, "test")

	// Check that app depends on db and redis
	assert.Contains(t, appUnit.Systemd.After, "test-db.service")
	assert.Contains(t, appUnit.Systemd.Requires, "test-db.service")
	assert.Contains(t, appUnit.Systemd.After, "test-redis.service")
	assert.Contains(t, appUnit.Systemd.Requires, "test-redis.service")

	// Check that web depends on app
	assert.Contains(t, webUnit.Systemd.After, "test-app.service")
	assert.Contains(t, webUnit.Systemd.Requires, "test-app.service")

	// Check that db has app as PartOf
	assert.Contains(t, dbUnit.Systemd.PartOf, "test-app.service")

	// Check that app has web as PartOf
	assert.Contains(t, appUnit.Systemd.PartOf, "test-web.service")
}

func TestFindTopLevelDependentService(t *testing.T) {
	// Create a dependency tree
	dependencyTree := map[string]*ServiceDependency{
		"db": {
			Dependencies:      map[string]struct{}{},
			DependentServices: map[string]struct{}{"app": {}},
		},
		"redis": {
			Dependencies:      map[string]struct{}{},
			DependentServices: map[string]struct{}{"app": {}},
		},
		"app": {
			Dependencies:      map[string]struct{}{"db": {}, "redis": {}},
			DependentServices: map[string]struct{}{"web": {}},
		},
		"web": {
			Dependencies:      map[string]struct{}{"app": {}},
			DependentServices: map[string]struct{}{},
		},
	}

	// Find the top-level dependent service for each service
	dbTopLevel := FindTopLevelDependentService("db", dependencyTree)
	redisTopLevel := FindTopLevelDependentService("redis", dependencyTree)
	appTopLevel := FindTopLevelDependentService("app", dependencyTree)
	webTopLevel := FindTopLevelDependentService("web", dependencyTree)

	// The top-level dependent service for db and redis should be web
	assert.Equal(t, "web", dbTopLevel)
	assert.Equal(t, "web", redisTopLevel)

	// The top-level dependent service for app should be web
	assert.Equal(t, "web", appTopLevel)

	// Web has no dependent services, so it should be empty
	assert.Equal(t, "", webTopLevel)
}