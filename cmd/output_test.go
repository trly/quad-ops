package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrintOutput_JSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := OperationResult{
		Success: true,
		Message: "Test successful",
		Items:   []string{"item1", "item2"},
	}

	err := PrintOutput("json", data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify JSON output
	var result OperationResult
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, data.Success, result.Success)
	assert.Equal(t, data.Message, result.Message)
	assert.Equal(t, data.Items, result.Items)
}

func TestPrintOutput_YAML(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := OperationResult{
		Success: true,
		Message: "Test successful",
		Items:   []string{"item1", "item2"},
	}

	err := PrintOutput("yaml", data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify YAML output
	var result OperationResult
	err = yaml.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, data.Success, result.Success)
	assert.Equal(t, data.Message, result.Message)
	assert.Equal(t, data.Items, result.Items)
}

func TestPrintOutput_YML(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := PrintOutput("yml", data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify YAML output
	var result map[string]string
	err = yaml.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestPrintOutput_Text(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := PrintOutput("text", data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify text output contains the data
	output := buf.String()
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestPrintOutput_UnsupportedFormat(t *testing.T) {
	data := map[string]string{"key": "value"}
	err := PrintOutput("invalid", data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format: invalid")
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}

	err := printJSON(data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify JSON formatting (should be indented)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(42), result["count"])
}

func TestPrintYAML(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := CheckResultStructured{
		Name:        "test-check",
		Status:      "pass",
		Message:     "All good",
		Suggestions: []string{"suggestion1", "suggestion2"},
	}

	err := printYAML(data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify YAML output
	var result CheckResultStructured
	err = yaml.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, data.Name, result.Name)
	assert.Equal(t, data.Status, result.Status)
	assert.Equal(t, data.Message, result.Message)
	assert.Equal(t, data.Suggestions, result.Suggestions)
}

func TestPrintText(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := struct {
		Field1 string
		Field2 int
	}{
		Field1: "value",
		Field2: 123,
	}

	err := printText(data)
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	output := buf.String()
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "123")
}
