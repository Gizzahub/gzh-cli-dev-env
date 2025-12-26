// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package status

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestNewStatusTableFormatter(t *testing.T) {
	formatter := NewStatusTableFormatter(true)
	if formatter == nil {
		t.Fatal("NewStatusTableFormatter returned nil")
	}
	if !formatter.UseColor {
		t.Error("UseColor should be true")
	}
}

func TestStatusTableFormatter_Format(t *testing.T) {
	formatter := NewStatusTableFormatter(false)

	statuses := []ServiceStatus{
		{
			Name:   "aws",
			Status: StatusActive,
			Current: CurrentConfig{
				Profile: "default",
				Region:  "us-east-1",
			},
			Credentials: CredentialStatus{Valid: true},
		},
		{
			Name:   "gcp",
			Status: StatusInactive,
			Current: CurrentConfig{
				Project: "my-project",
			},
			Credentials: CredentialStatus{Valid: false},
		},
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Check output contains expected content
	if !strings.Contains(output, "aws") {
		t.Error("Output should contain 'aws'")
	}
	if !strings.Contains(output, "gcp") {
		t.Error("Output should contain 'gcp'")
	}
	if !strings.Contains(output, "Development Environment Status") {
		t.Error("Output should contain header")
	}
}

func TestStatusTableFormatter_FormatEmpty(t *testing.T) {
	formatter := NewStatusTableFormatter(false)

	output, err := formatter.Format([]ServiceStatus{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if output != "No services to display" {
		t.Errorf("Expected 'No services to display', got %v", output)
	}
}

func TestNewStatusJSONFormatter(t *testing.T) {
	formatter := NewStatusJSONFormatter(true)
	if formatter == nil {
		t.Fatal("NewStatusJSONFormatter returned nil")
	}
	if !formatter.Pretty {
		t.Error("Pretty should be true")
	}
}

func TestStatusJSONFormatter_Format(t *testing.T) {
	formatter := NewStatusJSONFormatter(true)

	statuses := []ServiceStatus{
		{
			Name:   "aws",
			Status: StatusActive,
		},
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Verify valid JSON
	var parsed []ServiceStatus
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 status, got %d", len(parsed))
	}
	if parsed[0].Name != "aws" {
		t.Errorf("Expected name 'aws', got %v", parsed[0].Name)
	}
}

func TestStatusJSONFormatter_FormatCompact(t *testing.T) {
	formatter := NewStatusJSONFormatter(false)

	statuses := []ServiceStatus{
		{Name: "aws", Status: StatusActive},
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Compact JSON should not have newlines within the array
	if strings.Contains(output, "\n  ") {
		t.Error("Compact JSON should not have indentation")
	}
}

func TestNewStatusYAMLFormatter(t *testing.T) {
	formatter := NewStatusYAMLFormatter()
	if formatter == nil {
		t.Fatal("NewStatusYAMLFormatter returned nil")
	}
}

func TestStatusYAMLFormatter_Format(t *testing.T) {
	formatter := NewStatusYAMLFormatter()

	statuses := []ServiceStatus{
		{
			Name:   "kubernetes",
			Status: StatusActive,
			Current: CurrentConfig{
				Context:   "minikube",
				Namespace: "default",
			},
		},
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Verify valid YAML
	var parsed []ServiceStatus
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid YAML: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 status, got %d", len(parsed))
	}
	if parsed[0].Current.Context != "minikube" {
		t.Errorf("Expected context 'minikube', got %v", parsed[0].Current.Context)
	}
}

func TestStatusTableFormatter_FormatDuration(t *testing.T) {
	formatter := &StatusTableFormatter{UseColor: false}

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "< 1 min"},
		{5 * time.Minute, "5 min"},
		{2 * time.Hour, "2 hour"},
		{48 * time.Hour, "2 days"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatter.formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
			}
		})
	}
}
