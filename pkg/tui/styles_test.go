// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"testing"
)

// TestStyles_NotNil tests that style variables are initialized.
func TestStyles_NotNil(t *testing.T) {
	// Test that TitleStyle is usable
	rendered := TitleStyle.Render("Test")
	if rendered == "" {
		t.Error("TitleStyle should render content")
	}

	// Test that ServiceActiveStyle is usable
	rendered = ServiceActiveStyle.Render("Active")
	if rendered == "" {
		t.Error("ServiceActiveStyle should render content")
	}

	// Test that ServiceInactiveStyle is usable
	rendered = ServiceInactiveStyle.Render("Inactive")
	if rendered == "" {
		t.Error("ServiceInactiveStyle should render content")
	}

	// Test that ServiceErrorStyle is usable
	rendered = ServiceErrorStyle.Render("Error")
	if rendered == "" {
		t.Error("ServiceErrorStyle should render content")
	}
}

// TestGetStatusIcon tests the status icon function.
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"active", "✅"},
		{"connected", "✅"},
		{"running", "✅"},
		{"inactive", "❌"},
		{"stopped", "❌"},
		{"warning", "⚠️"},
		{"error", "🔴"},
		{"unknown", "❓"},
		{"", "❓"},
	}

	for _, tt := range tests {
		if got := GetStatusIcon(tt.status); got != tt.expected {
			t.Errorf("GetStatusIcon(%q) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

// TestColors tests that color constants are defined.
func TestColors(t *testing.T) {
	colors := []struct {
		name  string
		color interface{}
	}{
		{"ColorPrimary", ColorPrimary},
		{"ColorSecondary", ColorSecondary},
		{"ColorSuccess", ColorSuccess},
		{"ColorWarning", ColorWarning},
		{"ColorError", ColorError},
		{"ColorText", ColorText},
	}

	for _, c := range colors {
		if c.color == nil {
			t.Errorf("%s should be defined", c.name)
		}
	}
}
