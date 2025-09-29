package cmd

// SystemValidator provides system validation capabilities for commands.
type SystemValidator interface {
	SystemRequirements() error
}
