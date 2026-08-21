// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

const (
	pkgName = "aws"
	// DefaultProfile is the default AWS profile name.
	DefaultProfile = "default"
	// CredentialsExpiredMsg is the message for expired credentials.
	CredentialsExpiredMsg = "Credentials invalid or expired"
	awsCLIConfigure       = "configure"
	awsCLIRegion          = "region"
)

// Checker implements status.ServiceChecker for AWS.
type Checker struct{}

// NewChecker creates a new AWS status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (a *Checker) Name() string {
	return pkgName
}

// CheckStatus checks AWS current status.
func (a *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        pkgName,
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if AWS CLI is available
	if !a.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "AWS CLI not found"
		return st, nil
	}

	// Get current profile
	profile := a.getCurrentProfile()
	if profile == "" {
		st.Status = status.StatusInactive
		st.Details["error"] = "No AWS profile configured"
		return st, nil
	}

	st.Current.Profile = profile

	// Get current region
	region := a.getCurrentRegion()
	st.Current.Region = region

	// Check credentials validity
	credStatus, err := a.checkCredentials(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["credential_error"] = err.Error()
		//nolint:nilerr // intentional: errors converted to status fields
		return st, nil
	}

	st.Credentials = *credStatus
	if credStatus.Valid {
		st.Status = status.StatusActive
	} else {
		st.Status = status.StatusInactive
	}

	return st, nil
}

// CheckHealth performs detailed health check for AWS.
func (a *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]any),
	}

	// Test STS GetCallerIdentity
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to call AWS STS: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "AWS credentials are valid and accessible"
	health.Details["caller_identity"] = string(output)

	return health, nil
}

// isCLIAvailable checks if AWS CLI is installed.
func (a *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("aws")
	return err == nil
}

// getCurrentProfile gets the current AWS profile.
// Order matches Switcher.GetCurrentState: AWS_PROFILE → state file → empty.
func (a *Checker) getCurrentProfile() string {
	return resolveActiveProfile()
}

// getCurrentRegion gets the current AWS region.
func (a *Checker) getCurrentRegion() string {
	// Check AWS_REGION environment variable
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}

	// Check AWS_DEFAULT_REGION environment variable
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Try to get from AWS config for the active profile
	args := []string{awsCLIConfigure, "get", awsCLIRegion}
	if profile := resolveActiveProfile(); profile != "" {
		args = append(args, "--profile", profile)
	}
	//nolint:noctx // function lacks context parameter; cannot use CommandContext
	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return "us-east-1" // Default fallback
}

// checkCredentials checks AWS credentials validity.
//
//nolint:unparam // intentional: error always nil; errors converted to status fields
func (a *Checker) checkCredentials(ctx context.Context) (*status.CredentialStatus, error) {
	credStatus := &status.CredentialStatus{
		Valid: false,
		Type:  "aws-credentials",
	}

	// Test credentials with a simple STS call
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity")
	err := cmd.Run()
	if err != nil {
		credStatus.Warning = CredentialsExpiredMsg
		//nolint:nilerr // intentional: errors converted to status fields
		return credStatus, nil
	}

	credStatus.Valid = true

	// Try to get session token expiration (for assumed roles)
	cmd = exec.CommandContext(ctx, "aws", "sts", "get-session-token", "--duration-seconds", "900")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Parse session token response to get expiration
		// This is a simplified check - in practice you'd parse the JSON
		credStatus.Type = "session-token"
	}

	return credStatus, nil
}
