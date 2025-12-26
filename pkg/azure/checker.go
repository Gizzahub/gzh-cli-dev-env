// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// Checker implements status.ServiceChecker for Microsoft Azure.
type Checker struct{}

// NewChecker creates a new Azure status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (a *Checker) Name() string {
	return "azure"
}

// CheckStatus checks Azure current status.
func (a *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        "azure",
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if Azure CLI is available
	if !a.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "Azure CLI not found"
		return st, nil
	}

	// Get current subscription
	subscription, err := a.getCurrentSubscription(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["error"] = fmt.Sprintf("Failed to get current subscription: %v", err)
		return st, nil
	}

	if subscription == "" {
		st.Status = status.StatusInactive
		st.Details["error"] = "No Azure subscription configured"
		return st, nil
	}

	st.Current.Project = subscription

	// Get current account
	account, err := a.getCurrentAccount(ctx)
	if err == nil {
		st.Current.Account = account
	}

	// Check credentials validity
	credStatus, err := a.checkCredentials(ctx)
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

// CheckHealth performs detailed health check for Azure.
func (a *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]interface{}),
	}

	// Test Azure connectivity with az account show
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--output", "json")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to check Azure authentication: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "Azure credentials are valid and accessible"
	health.Details["account_info"] = string(output)

	return health, nil
}

// isCLIAvailable checks if Azure CLI is installed.
func (a *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("az")
	return err == nil
}

// getCurrentSubscription gets the current Azure subscription.
func (a *Checker) getCurrentSubscription(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--query", "name", "--output", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentAccount gets the current Azure account.
func (a *Checker) getCurrentAccount(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--query", "user.name", "--output", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// checkCredentials checks Azure credentials validity.
func (a *Checker) checkCredentials(ctx context.Context) (*status.CredentialStatus, error) {
	credStatus := &status.CredentialStatus{
		Valid: false,
		Type:  "azure-credentials",
	}

	// Test credentials with az account show
	cmd := exec.CommandContext(ctx, "az", "account", "show")
	err := cmd.Run()
	if err != nil {
		credStatus.Warning = "Credentials invalid or expired"
		return credStatus, nil
	}

	credStatus.Valid = true

	// Check authentication method
	cmd = exec.CommandContext(ctx, "az", "account", "show", "--query", "user.type", "--output", "tsv")
	output, err := cmd.Output()
	if err == nil {
		userType := strings.TrimSpace(string(output))
		switch userType {
		case "user":
			credStatus.Type = "user-account"
		case "servicePrincipal":
			credStatus.Type = "service-principal"
		default:
			credStatus.Type = userType
		}
	}

	return credStatus, nil
}
