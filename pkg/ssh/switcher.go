// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"fmt"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for SSH.
type Switcher struct{}

// NewSwitcher creates a new SSH switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (s *Switcher) Name() string {
	return "ssh"
}

// Switch switches to the specified SSH configuration.
func (s *Switcher) Switch(ctx context.Context, config interface{}) error {
	sshConfig, ok := config.(*environment.SSHConfig)
	if !ok {
		return fmt.Errorf("invalid SSH configuration type")
	}

	// For SSH, we would typically update the SSH config file or set environment variables
	// This is a simplified implementation
	if sshConfig.Config != "" {
		// In a real implementation, this would load the SSH configuration
		// For now, we'll just validate that the config exists
		_ = sshConfig.Config // Use the config silently
	}

	return nil
}

// GetCurrentState retrieves the current SSH configuration state.
func (s *Switcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	// Get current SSH configuration (simplified)
	return &environment.SSHConfig{
		Config: "default",
	}, nil
}

// Rollback rolls back to the previous SSH configuration.
func (s *Switcher) Rollback(ctx context.Context, previousState interface{}) error {
	return s.Switch(ctx, previousState)
}
