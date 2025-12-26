// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for Azure.
type Switcher struct{}

// NewSwitcher creates a new Azure switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (a *Switcher) Name() string {
	return "azure"
}

// Switch switches to the specified Azure configuration.
func (a *Switcher) Switch(ctx context.Context, config interface{}) error {
	azureConfig, ok := config.(*environment.AzureConfig)
	if !ok {
		return fmt.Errorf("invalid Azure configuration type")
	}

	// Set Azure subscription
	if azureConfig.Subscription != "" {
		cmd := exec.CommandContext(ctx, "az", "account", "set", "--subscription", azureConfig.Subscription)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Azure subscription: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current Azure configuration state.
func (a *Switcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	// Get current Azure subscription
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--query", "id", "-o", "tsv")
	subscriptionOutput, _ := cmd.Output()

	// Get current Azure tenant
	cmd = exec.CommandContext(ctx, "az", "account", "show", "--query", "tenantId", "-o", "tsv")
	tenantOutput, _ := cmd.Output()

	return &environment.AzureConfig{
		Subscription: strings.TrimSpace(string(subscriptionOutput)),
		Tenant:       strings.TrimSpace(string(tenantOutput)),
	}, nil
}

// Rollback rolls back to the previous Azure configuration.
func (a *Switcher) Rollback(ctx context.Context, previousState interface{}) error {
	return a.Switch(ctx, previousState)
}
