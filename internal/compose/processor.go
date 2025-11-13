// Package compose provides Docker Compose project processing functionality
package compose

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/service"
)

// SpecProcessor processes Docker Compose projects into service specs.
type SpecProcessor struct{}

// NewSpecProcessor creates a new SpecProcessor.
func NewSpecProcessor() *SpecProcessor {
	return &SpecProcessor{}
}

// Process converts a Docker Compose project to service specs.
// Uses project.WorkingDir for env file discovery.
func (p *SpecProcessor) Process(_ context.Context, project *types.Project) ([]service.Spec, error) {
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}

	// Use project's working directory for env file discovery
	converter := NewConverter(project.WorkingDir)
	return converter.ConvertProject(project)
}
