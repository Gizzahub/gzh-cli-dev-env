// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// commandContext builds an exec.Cmd. Overridable in tests to avoid real az CLI.
var commandContext = exec.CommandContext

// lookPath locates an executable. Overridable in tests.
var lookPath = exec.LookPath

// Switcher implements environment.ServiceSwitcher for Azure.
type Switcher struct{}

// NewSwitcher creates a new Azure switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (a *Switcher) Name() string {
	return azureName
}

// Switch switches to the specified Azure configuration.
func (a *Switcher) Switch(ctx context.Context, config any) error {
	azureConfig, ok := config.(*environment.AzureConfig)
	if !ok || azureConfig == nil {
		return fmt.Errorf("invalid Azure configuration type")
	}

	// Set Azure subscription
	if azureConfig.Subscription != "" {
		cmd := commandContext(ctx, "az", "account", "set", "--subscription", azureConfig.Subscription)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Azure subscription: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current Azure configuration state.
func (a *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	return currentStateWithCommand(ctx, commandContext)
}

func currentStateWithCommand(ctx context.Context, build func(context.Context, string, ...string) *exec.Cmd) (any, error) {
	subscriptionOutput, err := stateQuery(ctx, build, "subscription", "account", "show", "--query", "id", "-o", "tsv")
	if err != nil {
		return nil, err
	}
	tenantOutput, err := stateQuery(ctx, build, "tenant", "account", "show", "--query", "tenantId", "-o", "tsv")
	if err != nil {
		return nil, err
	}
	return &environment.AzureConfig{
		Subscription: strings.TrimSpace(string(subscriptionOutput)),
		Tenant:       strings.TrimSpace(string(tenantOutput)),
	}, nil
}

// 각 query 실패 시 부분 상태를 반환하지 않고 실행 오류와 취소 원인을 보존한다.
func stateQuery(ctx context.Context, build func(context.Context, string, ...string) *exec.Cmd, query string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("azure %s query: %w", query, err)
	}
	output, err := build(ctx, "az", args...).Output()
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("azure %s query: %w", query, err)
	}
	return output, nil
}

// Rollback rolls back to the previous Azure configuration.
func (a *Switcher) Rollback(ctx context.Context, previousState any) error {
	return a.Switch(ctx, previousState)
}
