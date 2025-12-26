// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package kubernetes

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
	if got := switcher.Name(); got != "kubernetes" {
		t.Errorf("Name() = %q, want %q", got, "kubernetes")
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

	err := switcher.Switch(ctx, "invalid-config")
	if err == nil {
		t.Error("Switch() with invalid config should return error")
	}

	if err.Error() != "invalid Kubernetes configuration type" {
		t.Errorf("Switch() error = %q, want %q", err.Error(), "invalid Kubernetes configuration type")
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

	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	if state == nil {
		t.Fatal("GetCurrentState() returned nil")
	}

	k8sConfig, ok := state.(*environment.KubernetesConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.KubernetesConfig", state)
	}

	_ = k8sConfig.Context
	_ = k8sConfig.Namespace
}

// TestSwitcher_Rollback_InvalidState tests rollback with invalid state type.
func TestSwitcher_Rollback_InvalidState(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := switcher.Rollback(ctx, "invalid-state")
	if err == nil {
		t.Error("Rollback() with invalid state should return error")
	}
}

// TestSwitcher_Switch_EmptyContext tests Switch with empty context name.
func TestSwitcher_Switch_EmptyContext(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty context should not cause an error (just skip the kubectl command)
	config := &environment.KubernetesConfig{
		Context:   "",
		Namespace: "",
	}

	err := switcher.Switch(ctx, config)
	// Should succeed since empty context means no action
	if err != nil {
		t.Logf("Switch() with empty context error = %v", err)
	}
}

// TestSwitcher_Switch_ValidConfig tests Switch with valid config structure.
func TestSwitcher_Switch_ValidConfig(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &environment.KubernetesConfig{
		Context:   "test-context",
		Namespace: "test-namespace",
	}

	err := switcher.Switch(ctx, config)
	// May fail if kubectl is not installed, but should not panic
	if err != nil {
		t.Logf("Switch() with valid config error (expected if kubectl not installed) = %v", err)
	}
}

// TestSwitcher_Rollback_WithValidState tests Rollback with valid state.
func TestSwitcher_Rollback_WithValidState(t *testing.T) {
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
		t.Logf("Rollback() with current state error (expected if kubectl not installed) = %v", err)
	}
}

// TestSwitcher_GetCurrentState_ReturnsKubernetesConfig tests return type.
func TestSwitcher_GetCurrentState_ReturnsKubernetesConfig(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	k8sConfig, ok := state.(*environment.KubernetesConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.KubernetesConfig", state)
	}

	// Log the current state for debugging
	t.Logf("Current Kubernetes context: %s", k8sConfig.Context)
	t.Logf("Current Kubernetes namespace: %s", k8sConfig.Namespace)
}
