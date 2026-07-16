// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"os"
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
	if got := checker.Name(); got != "aws" {
		t.Errorf("Name() = %q, want %q", got, "aws")
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

	if st.Name != "aws" {
		t.Errorf("status.Name = %q, want %q", st.Name, "aws")
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

// TestChecker_getCurrentProfile_WithEnvVar tests profile detection from environment.
func TestChecker_getCurrentProfile_WithEnvVar(t *testing.T) {
	checker := NewChecker()

	// Set AWS_PROFILE environment variable
	t.Setenv("AWS_PROFILE", "test-profile")

	profile := checker.getCurrentProfile()
	if profile != "test-profile" {
		t.Errorf("getCurrentProfile() = %q, want %q", profile, "test-profile")
	}
}

// TestChecker_getCurrentProfile_FromStateFile_whenEnvEmpty matches switcher resolution order.
func TestChecker_getCurrentProfile_FromStateFile_whenEnvEmpty(t *testing.T) {
	// Given
	dir := t.TempDir()
	orig := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = orig })
	t.Setenv("AWS_PROFILE", "")
	if err := writeActiveProfile("checker-state-profile"); err != nil {
		t.Fatalf("writeActiveProfile: %v", err)
	}
	checker := NewChecker()

	// When
	profile := checker.getCurrentProfile()

	// Then
	if profile != "checker-state-profile" {
		t.Errorf("getCurrentProfile() = %q, want %q", profile, "checker-state-profile")
	}
}

// TestChecker_getCurrentProfile_Empty_whenNeitherSet returns empty without default fallback.
func TestChecker_getCurrentProfile_Empty_whenNeitherSet(t *testing.T) {
	// Given
	dir := t.TempDir()
	orig := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = orig })
	t.Setenv("AWS_PROFILE", "")
	checker := NewChecker()

	// When
	profile := checker.getCurrentProfile()

	// Then
	if profile != "" {
		t.Errorf("getCurrentProfile() = %q, want empty", profile)
	}
}

// TestChecker_getCurrentRegion_WithEnvVar tests region detection from environment.
func TestChecker_getCurrentRegion_WithEnvVar(t *testing.T) {
	checker := NewChecker()

	// Save original values
	originalRegion := os.Getenv("AWS_REGION")
	originalDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_DEFAULT_REGION", originalDefaultRegion)
	}()

	// Clear both to test priority
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_DEFAULT_REGION")

	// Test AWS_REGION takes priority
	testRegion := "ap-northeast-2"
	os.Setenv("AWS_REGION", testRegion)

	region := checker.getCurrentRegion()
	if region != testRegion {
		t.Errorf("getCurrentRegion() = %q, want %q", region, testRegion)
	}
}

// TestChecker_getCurrentRegion_WithDefaultEnvVar tests AWS_DEFAULT_REGION fallback.
func TestChecker_getCurrentRegion_WithDefaultEnvVar(t *testing.T) {
	checker := NewChecker()

	// Save original values
	originalRegion := os.Getenv("AWS_REGION")
	originalDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_DEFAULT_REGION", originalDefaultRegion)
	}()

	// Clear AWS_REGION, set AWS_DEFAULT_REGION
	os.Unsetenv("AWS_REGION")
	testRegion := "eu-west-1"
	os.Setenv("AWS_DEFAULT_REGION", testRegion)

	region := checker.getCurrentRegion()
	if region != testRegion {
		t.Errorf("getCurrentRegion() = %q, want %q", region, testRegion)
	}
}

// TestConstants tests package constants.
func TestConstants(t *testing.T) {
	if DefaultProfile != "default" {
		t.Errorf("DefaultProfile = %q, want %q", DefaultProfile, "default")
	}

	if CredentialsExpiredMsg != "Credentials invalid or expired" {
		t.Errorf("CredentialsExpiredMsg = %q, unexpected value", CredentialsExpiredMsg)
	}
}
