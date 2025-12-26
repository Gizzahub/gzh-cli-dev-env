// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// Checker implements status.ServiceChecker for Google Cloud Platform.
type Checker struct{}

// NewChecker creates a new GCP status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (g *Checker) Name() string {
	return "gcp"
}

// CheckStatus checks GCP current status.
func (g *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        "gcp",
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if gcloud CLI is available
	if !g.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "gcloud CLI not found"
		return st, nil
	}

	// Get current project
	project, err := g.getCurrentProject(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["error"] = fmt.Sprintf("Failed to get current project: %v", err)
		return st, nil
	}

	if project == "" {
		st.Status = status.StatusInactive
		st.Details["error"] = "No GCP project configured"
		return st, nil
	}

	st.Current.Project = project

	// Get current account
	account, err := g.getCurrentAccount(ctx)
	if err == nil {
		st.Current.Account = account
	}

	// Get current region
	region, err := g.getCurrentRegion(ctx)
	if err == nil {
		st.Current.Region = region
	}

	// Check credentials validity
	credStatus, err := g.checkCredentials(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["credential_error"] = err.Error()
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

// CheckHealth performs detailed health check for GCP.
func (g *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]interface{}),
	}

	// Test GCP connectivity with gcloud auth list
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "list", "--format=json")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to check GCP authentication: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "GCP credentials are valid and accessible"
	health.Details["auth_list"] = string(output)

	return health, nil
}

// isCLIAvailable checks if gcloud CLI is installed.
func (g *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("gcloud")
	return err == nil
}

// getCurrentProject gets the current GCP project.
func (g *Checker) getCurrentProject(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentAccount gets the current GCP account.
func (g *Checker) getCurrentAccount(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentRegion gets the current GCP region.
func (g *Checker) getCurrentRegion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "compute/region")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// checkCredentials checks GCP credentials validity.
func (g *Checker) checkCredentials(ctx context.Context) (*status.CredentialStatus, error) {
	credStatus := &status.CredentialStatus{
		Valid: false,
		Type:  "gcp-credentials",
	}

	// Test credentials with gcloud auth application-default print-access-token
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
	err := cmd.Run()
	if err != nil {
		credStatus.Warning = "Credentials invalid or expired"
		return credStatus, nil
	}

	credStatus.Valid = true

	// Check if using service account
	cmd = exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
	output, err := cmd.Output()
	if err == nil {
		account := strings.TrimSpace(string(output))
		if strings.Contains(account, ".iam.gserviceaccount.com") {
			credStatus.Type = "service-account"
		} else {
			credStatus.Type = "user-account"
		}
	}

	return credStatus, nil
}
