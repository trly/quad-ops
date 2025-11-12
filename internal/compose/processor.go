// Package compose provides Docker Compose project processing functionality
package compose

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/service"
)

// SpecProcessor processes Docker Compose projects into service specs.
// It wraps Converter to provide the standard Process interface.
type SpecProcessor struct {
	converter *Converter
}

// NewSpecProcessor creates a new SpecProcessor with the given working directory.
func NewSpecProcessor(workingDir string) *SpecProcessor {
	return &SpecProcessor{
		converter: NewConverter(workingDir),
	}
}

// Process converts a Docker Compose project to service specs.
func (p *SpecProcessor) Process(_ context.Context, project *types.Project) ([]service.Spec, error) {
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}

	return p.converter.ConvertProject(project)
}
