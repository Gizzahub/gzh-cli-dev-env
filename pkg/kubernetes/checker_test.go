// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package kubernetes

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
	if got := checker.Name(); got != "kubernetes" {
		t.Errorf("Name() = %q, want %q", got, "kubernetes")
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

	if st.Name != "kubernetes" {
		t.Errorf("status.Name = %q, want %q", st.Name, "kubernetes")
	}

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

// TestChecker_CheckHealth_HasTimestamp tests CheckHealth sets timestamp correctly.
func TestChecker_CheckHealth_HasTimestamp(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	before := time.Now()
	health, err := checker.CheckHealth(ctx)
	after := time.Now()

	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	if health.CheckedAt.Before(before) || health.CheckedAt.After(after) {
		t.Errorf("health.CheckedAt = %v, should be between %v and %v", health.CheckedAt, before, after)
	}

	if health.Duration <= 0 {
		t.Errorf("health.Duration = %v, should be positive", health.Duration)
	}
}

// TestChecker_CheckStatus_DetailsNotNil tests that details map is initialized.
func TestChecker_CheckStatus_DetailsNotNil(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := checker.CheckStatus(ctx)
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}

	if st.Details == nil {
		t.Error("status.Details should not be nil")
	}
}

// TestChecker_CheckStatus_HasLastUsed tests that LastUsed is set.
func TestChecker_CheckStatus_HasLastUsed(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	before := time.Now()
	st, err := checker.CheckStatus(ctx)
	after := time.Now()

	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}

	if st.LastUsed.Before(before) || st.LastUsed.After(after) {
		t.Errorf("status.LastUsed = %v, should be between %v and %v", st.LastUsed, before, after)
	}
}

// TestChecker_isCLIAvailable tests CLI availability check.
func TestChecker_isCLIAvailable(t *testing.T) {
	checker := NewChecker()

	// This test just ensures isCLIAvailable doesn't panic
	// The result depends on whether kubectl is installed
	result := checker.isCLIAvailable()
	t.Logf("isCLIAvailable() = %v", result)
}

// TestChecker_CheckHealth_HasDetails tests that health details are populated.
func TestChecker_CheckHealth_HasDetails(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := checker.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	if health.Details == nil {
		t.Error("health.Details should not be nil")
	}
}

// TestChecker_CheckStatus_ContextCanceled tests behavior with canceled context.
func TestChecker_CheckStatus_ContextCanceled(t *testing.T) {
	checker := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	st, err := checker.CheckStatus(ctx)

	// Should return a valid status even with canceled context
	if err != nil {
		t.Logf("CheckStatus() with canceled context error = %v", err)
	}
	if st == nil {
		t.Error("CheckStatus() should return non-nil status even with canceled context")
	}
}
