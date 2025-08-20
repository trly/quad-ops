package compose

import (
	"github.com/compose-spec/compose-go/v2/types"
)

// ProcessProjects is a backward-compatible function that maintains the original API.
// It creates a processor instance and processes the projects.
func ProcessProjects(projects []*types.Project, force bool, existingProcessedUnits map[string]bool, doCleanup bool) (map[string]bool, error) {
	processor := NewDefaultProcessor(force)

	if existingProcessedUnits != nil {
		processor.WithExistingProcessedUnits(existingProcessedUnits)
	}

	err := processor.ProcessProjects(projects, doCleanup)
	if err != nil {
		return processor.GetProcessedUnits(), err
	}

	return processor.GetProcessedUnits(), nil
}
