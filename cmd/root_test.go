package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommandFlags verifies flag parsing.
func TestRootCommandFlags(t *testing.T) {
	rootCmd := &RootCommand{}
	cmd := rootCmd.GetCobraCommand()

	// Test flag defaults
	userFlag := cmd.PersistentFlags().Lookup("user")
	require.NotNil(t, userFlag)
	assert.Equal(t, "false", userFlag.DefValue)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)
}
