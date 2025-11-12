package cmd

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore HTTP/2 goroutines spawned by update command version checks
		// These are part of Go's standard HTTP client connection pooling
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}
