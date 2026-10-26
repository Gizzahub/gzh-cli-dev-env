// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// TestNewSwitcher verifies the constructor creates a valid switcher.
func TestNewSwitcher(t *testing.T) {
	switcher := NewSwitcher()
	if switcher == nil {
		t.Fatal("NewSwitcher() returned nil")
	}
}

// TestSwitcher_Name verifies the service name.
func TestSwitcher_Name(t *testing.T) {
	switcher := NewSwitcher()
	if got := switcher.Name(); got != "gcp" {
		t.Errorf("Name() = %q, want %q", got, "gcp")
	}
}

// TestSwitcher_ImplementsInterface verifies Switcher implements ServiceSwitcher.
func TestSwitcher_ImplementsInterface(t *testing.T) {
	var _ environment.ServiceSwitcher = (*Switcher)(nil)
}

// TestSwitcher_Switch_InvalidConfigType tests error handling for invalid config type.
func TestSwitcher_Switch_InvalidConfigType(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pass invalid config type
	err := switcher.Switch(ctx, "invalid-config")
	if err == nil {
		t.Error("Switch() with invalid config should return error")
	}

	// Check error message
	if err.Error() != "invalid GCP configuration type" {
		t.Errorf("Switch() error = %q, want %q", err.Error(), "invalid GCP configuration type")
	}
}

// TestSwitcher_Switch_NilConfig tests error handling for nil config.
func TestSwitcher_Switch_NilConfig(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := switcher.Switch(ctx, nil)
	if err == nil {
		t.Error("Switch() with nil config should return error")
	}
}

// TestSwitcher_GetCurrentState tests GetCurrentState returns valid structure.
func TestSwitcher_GetCurrentState(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := switcher.getCurrentState(ctx, successfulSnapshotOutput)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	if state == nil {
		t.Fatal("GetCurrentState() returned nil")
	}

	// Verify the state is the correct type
	gcpConfig, ok := state.(*environment.GCPConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.GCPConfig", state)
	}

	// GCPConfig should have Project, Account, and Region fields (can be empty)
	_ = gcpConfig.Project
	_ = gcpConfig.Account
	_ = gcpConfig.Region
}

// TestSwitcher_Rollback_InvalidState tests rollback with invalid state type.
func TestSwitcher_Rollback_InvalidState(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Rollback with invalid state should return error
	err := switcher.Rollback(ctx, "invalid-state")
	if err == nil {
		t.Error("Rollback() with invalid state should return error")
	}
}

// TestSwitcher_Switch_EmptyProject tests Switch with empty project.
func TestSwitcher_Switch_EmptyProject(t *testing.T) {
	trace := checkerFixture(t, map[string]checkerResponse{})
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty project should not cause an error (just skip the gcloud command)
	config := &environment.GCPConfig{
		Project: "",
		Account: "",
		Region:  "",
	}

	err := switcher.Switch(ctx, config)
	// Should succeed since empty project means no action
	if err != nil {
		t.Fatalf("Switch() with empty project error = %v", err)
	}
	assertCheckerCalls(t, trace)
}

// TestSwitcher_Switch_ValidConfig tests Switch with valid config structure.
func TestSwitcher_Switch_ValidConfig(t *testing.T) {
	trace := checkerFixture(t, map[string]checkerResponse{
		"config set project test-project-id":          {},
		"config set account test-account@example.com": {},
		"config set compute/region us-central1":       {},
	})
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &environment.GCPConfig{
		Project: "test-project-id",
		Account: "test-account@example.com",
		Region:  "us-central1",
	}

	err := switcher.Switch(ctx, config)
	// 소유 native fixture로만 공개 Switch 경로를 실행한다.
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	assertCheckerCalls(t, trace, "gcloud config set project test-project-id", "gcloud config set account test-account@example.com", "gcloud config set compute/region us-central1")
}

// TestSwitcher_Rollback_WithValidState tests Rollback with valid state.
func TestSwitcher_Rollback_WithValidState(t *testing.T) {
	responses := defaultCheckerResponses()
	responses["config set project project-1"] = checkerResponse{}
	responses["config set account user@example.invalid"] = checkerResponse{}
	responses["config set compute/region region-1"] = checkerResponse{}
	trace := checkerFixture(t, responses)
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current state first
	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	// Rollback to current state should work
	err = switcher.Rollback(ctx, state)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	assertCheckerCalls(t, trace, "gcloud config get-value project", "gcloud config get-value account", "gcloud config get-value compute/region", "gcloud config set project project-1", "gcloud config set account user@example.invalid", "gcloud config set compute/region region-1")
}

// TestSwitcher_GetCurrentState_ReturnsGCPConfig tests return type.
func TestSwitcher_GetCurrentState_ReturnsGCPConfig(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := switcher.getCurrentState(ctx, successfulSnapshotOutput)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	gcpConfig, ok := state.(*environment.GCPConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.GCPConfig", state)
	}

	// Log the current state for debugging
	t.Logf("Current GCP project: %s", gcpConfig.Project)
	t.Logf("Current GCP account: %s", gcpConfig.Account)
	t.Logf("Current GCP region: %s", gcpConfig.Region)
}
