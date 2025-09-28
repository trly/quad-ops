// Package cmd provides output formatting utilities for quad-ops CLI.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// OutputData represents structured data that can be output in multiple formats.
type OutputData struct {
	Data interface{} `json:"data,omitempty" yaml:"data,omitempty"`
}

// PrintOutput formats and prints data according to the specified output format.
func PrintOutput(format string, data interface{}) error {
	switch strings.ToLower(format) {
	case "json":
		return printJSON(data)
	case "yaml", "yml":
		return printYAML(data)
	case "text":
		return printText(data)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// printJSON outputs data as JSON.
func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML outputs data as YAML.
func printYAML(data interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer func() {
		_ = encoder.Close() // Ignore close error for stdout
	}()
	return encoder.Encode(data)
}

// printText outputs data in a human-readable text format.
func printText(data interface{}) error {
	// For text output, we expect the caller to handle formatting
	// This is a fallback that just prints the data
	fmt.Printf("%+v\n", data)
	return nil
}

// OperationResult represents the result of an operation that can be output in structured format.
type OperationResult struct {
	Success bool              `json:"success" yaml:"success"`
	Message string            `json:"message,omitempty" yaml:"message,omitempty"`
	Items   []string          `json:"items,omitempty" yaml:"items,omitempty"`
	Errors  []string          `json:"errors,omitempty" yaml:"errors,omitempty"`
	Details map[string]string `json:"details,omitempty" yaml:"details,omitempty"`
}

// CheckResultStructured represents a health check result in structured format.
type CheckResultStructured struct {
	Name        string   `json:"name" yaml:"name"`
	Status      string   `json:"status" yaml:"status"`
	Message     string   `json:"message,omitempty" yaml:"message,omitempty"`
	Suggestions []string `json:"suggestions,omitempty" yaml:"suggestions,omitempty"`
}

// HealthCheckOutput represents the output of the doctor command.
type HealthCheckOutput struct {
	Overall string                  `json:"overall" yaml:"overall"`
	Checks  []CheckResultStructured `json:"checks" yaml:"checks"`
	Summary map[string]int          `json:"summary" yaml:"summary"`
}
