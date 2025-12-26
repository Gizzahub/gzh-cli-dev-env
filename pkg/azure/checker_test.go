// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// TestNewChecker verifies the constructor creates a valid checker.
func TestNewChecker(t *testing.T) {
	checker := NewChecker()
	if checker == nil {
		t.Fatal("NewChecker() returned nil")
	}
}

// TestChecker_Name verifies the service name.
func TestChecker_Name(t *testing.T) {
	checker := NewChecker()
	if got := checker.Name(); got != "azure" {
		t.Errorf("Name() = %q, want %q", got, "azure")
	}
}

// TestChecker_ImplementsInterface verifies Checker implements ServiceChecker.
func TestChecker_ImplementsInterface(t *testing.T) {
	var _ status.ServiceChecker = (*Checker)(nil)
}

// TestChecker_CheckStatus_ReturnsValidStatus tests CheckStatus returns valid status structure.
func TestChecker_CheckStatus_ReturnsValidStatus(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := checker.CheckStatus(ctx)
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}

	if st == nil {
		t.Fatal("CheckStatus() returned nil status")
	}

	if st.Name != "azure" {
		t.Errorf("status.Name = %q, want %q", st.Name, "azure")
	}

	// Status should be one of the valid status types
	validStatuses := map[status.StatusType]bool{
		status.StatusActive:   true,
		status.StatusInactive: true,
		status.StatusError:    true,
		status.StatusUnknown:  true,
	}
	if !validStatuses[st.Status] {
		t.Errorf("status.Status = %v, not a valid status type", st.Status)
	}
}

// TestChecker_CheckHealth_ReturnsValidHealth tests CheckHealth returns valid health structure.
func TestChecker_CheckHealth_ReturnsValidHealth(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := checker.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	if health == nil {
		t.Fatal("CheckHealth() returned nil health")
	}

	// Health should have a valid status
	validStatuses := map[status.StatusType]bool{
		status.StatusActive:   true,
		status.StatusInactive: true,
		status.StatusError:    true,
		status.StatusUnknown:  true,
	}
	if !validStatuses[health.Status] {
		t.Errorf("health.Status = %v, not a valid status type", health.Status)
	}
}
