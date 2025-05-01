package systemd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/config"
)

func TestGetSystemdUnitType(t *testing.T) {
	// Save original config and restore it after the test
	origConfig := config.GetConfig()
	defer config.SetConfig(origConfig)

	testCases := []struct {
		name           string
		userMode       bool
		unitType       string
		expectedResult string
	}{
		{"System mode container", false, "container", "container"},
		{"System mode volume", false, "volume", "volume"},
		{"System mode network", false, "network", "network"},
		{"System mode service", false, "service", "service"},
		{"User mode container", true, "container", "service"},
		{"User mode volume", true, "volume", "service"},
		{"User mode network", true, "network", "service"},
		{"User mode service", true, "service", "service"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test config
			testConfig := &config.Config{
				UserMode: tc.userMode,
				Verbose:  false,
			}
			config.SetConfig(testConfig)

			result := GetSystemdUnitType(tc.unitType)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestValidateUnitNameAndType(t *testing.T) {
	// Valid cases
	tests := []struct {
		name     string
		unitType string
		valid    bool
	}{
		{"simple-service", "service", true},
		{"container-name", "container", true},
		{"my_volume", "volume", true},
		{"app-network", "network", true},
		{"invalid@char", "service", true}, // @ is actually valid in systemd
		{"123numeric", "service", true},
		{"UPPERCASE", "service", true},
		{"dash-ending-", "service", true},
		{"under_score", "service", true},
		{"with.dot", "service", true},

		// Invalid cases
		{"#invalid", "service", false}, // Invalid character
		{"", "service", false},         // Empty name
		{"valid-name", "UPPER", false}, // Invalid type (uppercase)
		{"valid-name", "type@", false}, // Invalid type (special char)
		{"valid-name", "", false},     // Empty type
	}

	for _, test := range tests {
		err := ValidateUnitNameAndType(test.name, test.unitType)
		if test.valid {
			assert.NoError(t, err, "Expected valid for name=%s, type=%s", test.name, test.unitType)
		} else {
			assert.Error(t, err, "Expected invalid for name=%s, type=%s", test.name, test.unitType)
		}
	}
}