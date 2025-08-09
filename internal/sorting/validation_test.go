package sorting

import (
	"testing"
)

func TestValidateUnitName(t *testing.T) {
	tests := []struct {
		name      string
		unitName  string
		expectErr bool
	}{
		{"valid unit name", "myapp.service", false},
		{"valid with underscore", "my_app.service", false},
		{"valid with dash", "my-app.service", false},
		{"valid with at symbol", "my@app.service", false},
		{"valid with colon", "my:app.service", false},
		{"valid with dot", "my.app.service", false},
		{"empty name", "", true},
		{"command injection semicolon", "app; rm -rf /", true},
		{"command injection pipe", "app | cat /etc/passwd", true},
		{"command injection ampersand", "app && rm file", true},
		{"command injection dollar", "app$USER", true},
		{"command injection backtick", "app`whoami`", true},
		{"too long name", string(make([]rune, 300)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUnitName(tt.unitName)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for unit name: %s", tt.unitName)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for unit name %s: %v", tt.unitName, err)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{"valid relative path", "app/config", false},
		{"valid absolute path", "/app/config", false},
		{"empty path", "", true},
		{"path traversal dotdot", "../etc/passwd", true},
		{"path traversal in middle", "app/../../../etc/passwd", true},
		{"path traversal at end", "app/..", true},
		{"normalized safe path", "app/./config", false},
		{"complex traversal", "app/../../etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for path: %s", tt.path)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for path %s: %v", tt.path, err)
			}
		})
	}
}

func TestValidatePathWithinBase(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		basePath  string
		expectErr bool
		expected  string
	}{
		{"valid relative path", "config.yaml", "/app", false, "/app/config.yaml"},
		{"valid nested path", "configs/app.yaml", "/app", false, "/app/configs/app.yaml"},
		{"path traversal attempt", "../../../etc/passwd", "/app", true, ""},
		{"absolute path within base", "/app/config.yaml", "/app", false, "/app/config.yaml"},
		{"absolute path outside base", "/etc/passwd", "/app", true, ""},
		{"empty path", "", "/app", true, ""},
		{"empty base", "config.yaml", "", true, ""},
		{"complex traversal", "configs/../../../etc/passwd", "/app", true, ""},
		{"normalized safe path", "configs/./app.yaml", "/app", false, "/app/configs/app.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePathWithinBase(tt.path, tt.basePath)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for path: %s, base: %s", tt.path, tt.basePath)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for path %s, base %s: %v", tt.path, tt.basePath, err)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("expected result %s, got %s", tt.expected, result)
			}
		})
	}
}
