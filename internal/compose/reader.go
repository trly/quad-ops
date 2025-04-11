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

	// Validate path before proceeding
	if path == "" {
		return nil, fmt.Errorf("empty compose directory path provided")
	}

	// Check if the directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("compose directory does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to access compose directory: %w", err)
	}

	// Ensure it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	if config.GetConfig().Verbose {
		log.Printf("reading docker-compose files from %s", path)
	}

	composeFilesFound := false

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but continue walking if possible
			if config.GetConfig().Verbose {
				log.Printf("error accessing path %s: %v", filePath, err)
			}
			return nil
		}

		if !info.IsDir() && (filepath.Ext(filePath) == ".yaml" || filepath.Ext(filePath) == ".yml") {
			// Check if the file name starts with docker-compose or compose
			baseName := filepath.Base(filePath)
			isComposeFile := false
			
			// Common Docker Compose file patterns
			if baseName == "docker-compose.yml" || baseName == "docker-compose.yaml" || 
			   baseName == "compose.yml" || baseName == "compose.yaml" ||
			   filepath.Dir(filePath) != path { // Any yaml in subdirectories
				isComposeFile = true
			}
			
			if isComposeFile {
				composeFilesFound = true
				project, err := ParseComposeFile(filePath)
				if err != nil {
					if config.GetConfig().Verbose {
						log.Printf("error parsing compose file %s: %v", filePath, err)
					}
					// Continue processing other files
					return nil
				}
				projects = append(projects, project)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose files: %w", err)
	}
	
	// No compose files found in the directory
	if !composeFilesFound {
		if config.GetConfig().Verbose {
			log.Printf("no docker-compose files found in %s", path)
		}
		// Return empty list instead of error, as this is not necessarily an error condition
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
