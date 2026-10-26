// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for Docker.
type Switcher struct{}

// NewSwitcher creates a new Docker switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (d *Switcher) Name() string {
	return serviceName
}

// Switch switches to the specified Docker configuration.
func (d *Switcher) Switch(ctx context.Context, config any) error {
	dockerConfig, ok := config.(*environment.DockerConfig)
	if !ok {
		return fmt.Errorf("invalid Docker configuration type")
	}

	// Set Docker context
	if dockerConfig.Context != "" {
		cmd := exec.CommandContext(ctx, "docker", "context", "use", dockerConfig.Context)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Docker context: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current Docker configuration state.
func (d *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	return d.getCurrentState(ctx, runSnapshotCommand)
}

// snapshotCommand는 호출별 실행 경계이며 새 전역 가변 hook을 만들지 않는다.
type snapshotCommand func(context.Context, string, ...string) ([]byte, error)

func (d *Switcher) getCurrentState(ctx context.Context, run snapshotCommand) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("docker state snapshot: %w", err)
	}
	currentContext, err := snapshotValue(ctx, run, "context", "context", "show")
	if err != nil {
		return nil, err
	}
	return &environment.DockerConfig{Context: currentContext}, nil
}

func runSnapshotCommand(ctx context.Context, _ string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	return cmd.Output()
}

func snapshotValue(ctx context.Context, run snapshotCommand, query string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("docker %s state query: %w", query, err)
	}
	output, err := run(ctx, "docker", args...)
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("docker %s state query: %w", query, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Rollback rolls back to the previous Docker configuration.
func (d *Switcher) Rollback(ctx context.Context, previousState any) error {
	return d.Switch(ctx, previousState)
}
