// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for GCP.
type Switcher struct{}

// NewSwitcher creates a new GCP switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (g *Switcher) Name() string {
	return "gcp"
}

// Switch switches to the specified GCP configuration.
func (g *Switcher) Switch(ctx context.Context, config interface{}) error {
	gcpConfig, ok := config.(*environment.GCPConfig)
	if !ok {
		return fmt.Errorf("invalid GCP configuration type")
	}

	// Set GCP project
	if gcpConfig.Project != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "project", gcpConfig.Project)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP project: %w", err)
		}
	}

	// Set GCP account
	if gcpConfig.Account != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "account", gcpConfig.Account)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP account: %w", err)
		}
	}

	// Set GCP region
	if gcpConfig.Region != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "compute/region", gcpConfig.Region)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP region: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current GCP configuration state.
func (g *Switcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	// Get current GCP project
	cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "project")
	projectOutput, _ := cmd.Output()

	// Get current GCP account
	cmd = exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
	accountOutput, _ := cmd.Output()

	// Get current GCP region
	cmd = exec.CommandContext(ctx, "gcloud", "config", "get-value", "compute/region")
	regionOutput, _ := cmd.Output()

	return &environment.GCPConfig{
		Project: strings.TrimSpace(string(projectOutput)),
		Account: strings.TrimSpace(string(accountOutput)),
		Region:  strings.TrimSpace(string(regionOutput)),
	}, nil
}

// Rollback rolls back to the previous GCP configuration.
func (g *Switcher) Rollback(ctx context.Context, previousState interface{}) error {
	return g.Switch(ctx, previousState)
}
