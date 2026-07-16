// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
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
	// Get current Docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	contextOutput, _ := cmd.Output() //nolint:errcheck // Error ignored; empty string used on failure

	return &environment.DockerConfig{
		Context: strings.TrimSpace(string(contextOutput)),
	}, nil
}

// Rollback rolls back to the previous Docker configuration.
func (d *Switcher) Rollback(ctx context.Context, previousState any) error {
	return d.Switch(ctx, previousState)
}
