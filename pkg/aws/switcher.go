// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for AWS.
type Switcher struct{}

// NewSwitcher creates a new AWS switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (a *Switcher) Name() string {
	return "aws"
}

// Switch switches to the specified AWS configuration.
func (a *Switcher) Switch(ctx context.Context, config interface{}) error {
	awsConfig, ok := config.(*environment.AWSConfig)
	if !ok {
		return fmt.Errorf("invalid AWS configuration type")
	}

	// Set AWS profile
	if awsConfig.Profile != "" {
		cmd := exec.CommandContext(ctx, "aws", "configure", "set", "profile", awsConfig.Profile)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set AWS profile: %w", err)
		}
	}

	// Set AWS region
	if awsConfig.Region != "" {
		args := []string{"configure", "set", "region", awsConfig.Region}
		if awsConfig.Profile != "" {
			args = append(args, "--profile", awsConfig.Profile)
		}
		cmd := exec.CommandContext(ctx, "aws", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set AWS region: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current AWS configuration state.
func (a *Switcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	// Get current AWS profile
	cmd := exec.CommandContext(ctx, "aws", "configure", "get", "profile")
	profileOutput, _ := cmd.Output()

	// Get current AWS region
	cmd = exec.CommandContext(ctx, "aws", "configure", "get", "region")
	regionOutput, _ := cmd.Output()

	return &environment.AWSConfig{
		Profile: strings.TrimSpace(string(profileOutput)),
		Region:  strings.TrimSpace(string(regionOutput)),
	}, nil
}

// Rollback rolls back to the previous AWS configuration.
func (a *Switcher) Rollback(ctx context.Context, previousState interface{}) error {
	return a.Switch(ctx, previousState)
}
