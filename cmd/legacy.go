// Package cmd contains legacy patterns that are being migrated.
//
// This file exists as a placeholder to track deprecated patterns during
// the migration to dependency injection. It should be removed once all
// commands have been successfully migrated.
//
// Deprecated patterns being removed:
// - Global variables for command state
// - Direct os.Exit() calls in command handlers
// - Global test seams for mocking system dependencies
package cmd
