// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"errors"
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
	return gcpName
}

// Switch switches to the specified GCP configuration.
func (g *Switcher) Switch(ctx context.Context, config any) error {
	gcpConfig, ok := config.(*environment.GCPConfig)
	if !ok || gcpConfig == nil {
		return fmt.Errorf("invalid GCP configuration type")
	}
	// Validate every value before running any command so a rejected field
	// never leaves a partially applied configuration.
	for _, field := range []struct{ name, value string }{
		{"GCP project", gcpConfig.Project},
		{"GCP account", gcpConfig.Account},
		{"GCP region", gcpConfig.Region},
	} {
		if err := environment.RejectOptionLikeValue(field.name, field.value); err != nil {
			return err
		}
	}

	// Set GCP project
	if gcpConfig.Project != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "project", gcpConfig.Project) // #nosec G204 -- fixed binary with argv (no shell); the value was checked above not to start with '-'
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP project: %w", err)
		}
	}

	// Set GCP account
	if gcpConfig.Account != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "account", gcpConfig.Account) // #nosec G204 -- fixed binary with argv (no shell); the value was checked above not to start with '-'
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP account: %w", err)
		}
	}

	// Set GCP region
	if gcpConfig.Region != "" {
		cmd := exec.CommandContext(ctx, "gcloud", "config", "set", "compute/region", gcpConfig.Region) // #nosec G204 -- fixed binary with argv (no shell); the value was checked above not to start with '-'
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set GCP region: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current GCP configuration state.
func (g *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	return g.getCurrentState(ctx, runSnapshotCommand)
}

// snapshotCommand는 호출별 실행 경계이며 새 전역 가변 hook을 만들지 않는다.
type snapshotCommand func(context.Context, string, ...string) ([]byte, error)

func (g *Switcher) getCurrentState(ctx context.Context, run snapshotCommand) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("gcp state snapshot: %w", err)
	}
	project, err := snapshotValue(ctx, run, "project", "config", "get-value", "project")
	if err != nil {
		return nil, err
	}
	account, err := snapshotValue(ctx, run, "account", "config", "get-value", "account")
	if err != nil {
		return nil, err
	}
	region, err := snapshotValue(ctx, run, "region", "config", "get-value", "compute/region")
	if err != nil {
		return nil, err
	}
	return &environment.GCPConfig{Project: project, Account: account, Region: region}, nil
}

func runSnapshotCommand(ctx context.Context, _ string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gcloud", args...) // #nosec G204 -- fixed binary with argv (no shell); args are fixed literals from snapshotValue callers
	return cmd.Output()
}

func snapshotValue(ctx context.Context, run snapshotCommand, query string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("gcp %s state query: %w", query, err)
	}
	output, err := run(ctx, "gcloud", args...)
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("gcp %s state query: %w", query, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Rollback rolls back to the previous GCP configuration.
func (g *Switcher) Rollback(ctx context.Context, previousState any) error {
	return g.Switch(ctx, previousState)
}
