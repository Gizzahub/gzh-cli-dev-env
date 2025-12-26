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
