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

// TestStatusTableFormatter_formatStatus tests all status type formatting.
func TestStatusTableFormatter_formatStatus(t *testing.T) {
	formatter := &StatusTableFormatter{UseColor: false}

	tests := []struct {
		status   StatusType
		contains string
	}{
		{StatusActive, "Active"},
		{StatusInactive, "Inactive"},
		{StatusError, "Error"},
		{StatusUnknown, "Unknown"},
		{StatusType("invalid"), "Unknown"}, // Default case
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := formatter.formatStatus(tt.status)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatStatus(%v) = %q, should contain %q", tt.status, result, tt.contains)
			}
		})
	}
}

// TestStatusTableFormatter_formatCredentials tests credential status formatting.
func TestStatusTableFormatter_formatCredentials(t *testing.T) {
	formatter := &StatusTableFormatter{UseColor: false}

	tests := []struct {
		name     string
		creds    CredentialStatus
		contains string
	}{
		{
			name:     "invalid credentials",
			creds:    CredentialStatus{Valid: false},
			contains: "Invalid",
		},
		{
			name:     "warning with expire",
			creds:    CredentialStatus{Valid: true, Warning: "will expire soon"},
			contains: "Expires",
		},
		{
			name:     "warning without expire",
			creds:    CredentialStatus{Valid: true, Warning: "some warning"},
			contains: "Warning",
		},
		{
			name: "expires in less than 24 hours",
			creds: CredentialStatus{
				Valid:     true,
				ExpiresAt: time.Now().Add(12 * time.Hour),
			},
			contains: "hour",
		},
		{
			name: "expires in more than 24 hours",
			creds: CredentialStatus{
				Valid:     true,
				ExpiresAt: time.Now().Add(48 * time.Hour),
			},
			contains: "days",
		},
		{
			name:     "valid without expiry",
			creds:    CredentialStatus{Valid: true},
			contains: "Valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.formatCredentials(tt.creds)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatCredentials() = %q, should contain %q", result, tt.contains)
			}
		})
	}
}

// TestStatusTableFormatter_formatLastUsed tests last used time formatting.
func TestStatusTableFormatter_formatLastUsed(t *testing.T) {
	formatter := &StatusTableFormatter{UseColor: false}

	// Test zero time
	result := formatter.formatLastUsed(time.Time{})
	if result != "Unknown" {
		t.Errorf("formatLastUsed(zero) = %q, want %q", result, "Unknown")
	}

	// Test recent time
	result = formatter.formatLastUsed(time.Now().Add(-5 * time.Minute))
	if !strings.Contains(result, "ago") {
		t.Errorf("formatLastUsed() = %q, should contain 'ago'", result)
	}
}

// TestStatusTableFormatter_formatCurrent tests current config formatting.
func TestStatusTableFormatter_formatCurrent(t *testing.T) {
	formatter := &StatusTableFormatter{UseColor: false}

	tests := []struct {
		name     string
		current  CurrentConfig
		expected string
	}{
		{
			name:     "empty config",
			current:  CurrentConfig{},
			expected: "-",
		},
		{
			name:     "profile only",
			current:  CurrentConfig{Profile: "default"},
			expected: "default",
		},
		{
			name:     "with region",
			current:  CurrentConfig{Profile: "prod", Region: "us-west-2"},
			expected: "prod (us-west-2)",
		},
		{
			name:     "with namespace",
			current:  CurrentConfig{Context: "k8s", Namespace: "production"},
			expected: "k8s /production",
		},
		{
			name:     "default namespace ignored",
			current:  CurrentConfig{Context: "k8s", Namespace: "default"},
			expected: "k8s",
		},
		{
			name:     "long string truncated",
			current:  CurrentConfig{Profile: "very-long-profile-name-that-exceeds-limit"},
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.formatCurrent(tt.current)
			if tt.expected == "..." {
				if !strings.HasSuffix(result, "...") {
					t.Errorf("formatCurrent() = %q, should end with '...'", result)
				}
			} else if result != tt.expected {
				t.Errorf("formatCurrent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestStatusTableFormatter_colorize tests colorization.
func TestStatusTableFormatter_colorize(t *testing.T) {
	// Test without color
	formatterNoColor := &StatusTableFormatter{UseColor: false}
	result := formatterNoColor.colorize("test", "green")
	if result != "test" {
		t.Errorf("colorize without UseColor = %q, want %q", result, "test")
	}

	// Test with color
	formatterWithColor := &StatusTableFormatter{UseColor: true}

	// Test valid colors
	colors := []string{"red", "green", "yellow", "gray"}
	for _, color := range colors {
		result := formatterWithColor.colorize("test", color)
		if !strings.Contains(result, "\033[") {
			t.Errorf("colorize with %s should contain ANSI escape code", color)
		}
		if !strings.Contains(result, "test") {
			t.Errorf("colorize should contain original text")
		}
	}

	// Test unknown color
	result = formatterWithColor.colorize("test", "unknown")
	if result != "test" {
		t.Errorf("colorize with unknown color = %q, want %q", result, "test")
	}
}

// TestStatusTableFormatter_FormatWithColor tests table formatting with colors.
func TestStatusTableFormatter_FormatWithColor(t *testing.T) {
	formatter := NewStatusTableFormatter(true) // With color

	statuses := []ServiceStatus{
		{
			Name:   "aws",
			Status: StatusActive,
			Current: CurrentConfig{
				Profile: "default",
			},
			Credentials: CredentialStatus{Valid: true},
		},
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Should contain ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Error("Output with UseColor should contain ANSI escape codes")
	}
}
