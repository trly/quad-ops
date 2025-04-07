package compose

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
)

func ReadProjects(path string) ([]*types.Project, error) {
	var projects []*types.Project

	if config.GetConfig().Verbose {
		log.Printf("reading docker-compose files from %s", path)
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			project, err := ParseComposeFile(path)
			if err != nil {
				return err
			}
			projects = append(projects, project)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose files: %w", err)
	}

	return projects, nil
}

func ParseComposeFile(path string) (*types.Project, error) {
	ctx := context.Background()
	projectName := loader.NormalizeProjectName(filepath.Dir(path))

	options, err := cli.NewProjectOptions(
		[]string{path},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName(projectName),
	)

	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(ctx, options)
	if err != nil {
		return nil, err
	}

	return project, nil
}
