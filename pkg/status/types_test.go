// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package status

import (
	"testing"
	"time"
)

func TestStatusType_String(t *testing.T) {
	tests := []struct {
		status   StatusType
		expected string
	}{
		{StatusActive, "active"},
		{StatusInactive, "inactive"},
		{StatusError, "error"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := string(tt.status); got != tt.expected {
				t.Errorf("StatusType = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestServiceStatus_Fields(t *testing.T) {
	now := time.Now()
	status := ServiceStatus{
		Name:   "aws",
		Status: StatusActive,
		Current: CurrentConfig{
			Profile: "default",
			Region:  "us-east-1",
		},
		Credentials: CredentialStatus{
			Valid:     true,
			ExpiresAt: now.Add(1 * time.Hour),
		},
		LastUsed: now,
	}

	if status.Name != "aws" {
		t.Errorf("Name = %v, want aws", status.Name)
	}
	if status.Status != StatusActive {
		t.Errorf("Status = %v, want active", status.Status)
	}
	if status.Current.Profile != "default" {
		t.Errorf("Current.Profile = %v, want default", status.Current.Profile)
	}
	if !status.Credentials.Valid {
		t.Error("Credentials.Valid should be true")
	}
}

func TestCurrentConfig_Empty(t *testing.T) {
	config := CurrentConfig{}

	if config.Profile != "" {
		t.Error("Empty CurrentConfig.Profile should be empty string")
	}
	if config.Region != "" {
		t.Error("Empty CurrentConfig.Region should be empty string")
	}
}

func TestCredentialStatus_Expired(t *testing.T) {
	expired := CredentialStatus{
		Valid:     true,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	if time.Until(expired.ExpiresAt) > 0 {
		t.Error("Credential should be expired")
	}
}

func TestHealthStatus_Fields(t *testing.T) {
	health := HealthStatus{
		Status:    StatusActive,
		Message:   "All systems operational",
		CheckedAt: time.Now(),
		Duration:  100 * time.Millisecond,
		Details:   map[string]interface{}{"api": "ok"},
	}

	if health.Status != StatusActive {
		t.Error("Status should be active")
	}
	if health.Message != "All systems operational" {
		t.Errorf("Message = %v, want 'All systems operational'", health.Message)
	}
	if health.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", health.Duration)
	}
}

func TestStatusOptions_Defaults(t *testing.T) {
	opts := StatusOptions{}

	if opts.Parallel {
		t.Error("Default Parallel should be false")
	}
	if opts.CheckHealth {
		t.Error("Default CheckHealth should be false")
	}
	if opts.Timeout != 0 {
		t.Error("Default Timeout should be zero")
	}
}
