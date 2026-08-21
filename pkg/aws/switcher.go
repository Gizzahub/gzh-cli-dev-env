// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// commandContext builds an exec.Cmd. Overridable in tests to capture argv.
var commandContext = exec.CommandContext

// Switcher implements environment.ServiceSwitcher for AWS.
type Switcher struct{}

// NewSwitcher creates a new AWS switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (a *Switcher) Name() string {
	return pkgName
}

// Switch activates the specified AWS configuration.
//
// Profile activation is persisted to a state file under the user config dir
// (not via `aws configure set profile`, which does not change the active
// profile). Callers that spawn child processes should also export AWS_PROFILE
// so the AWS CLI/SDK pick up the same profile.
//
// Region is written with `aws configure set region` scoped to --profile when
// a profile is provided.
func (a *Switcher) Switch(ctx context.Context, config any) error {
	awsConfig, ok := config.(*environment.AWSConfig)
	if !ok {
		return fmt.Errorf("invalid AWS configuration type")
	}

	// Persist active profile. This is the real activation signal for this tool.
	// Do NOT use `aws configure set profile` — that only writes a key into the
	// default profile block and does not change which profile is active.
	if awsConfig.Profile != "" {
		if err := writeActiveProfile(awsConfig.Profile); err != nil {
			return fmt.Errorf("failed to set AWS profile: %w", err)
		}
	}

	// Set region on the target profile (or default when profile is empty).
	if awsConfig.Region != "" {
		args := []string{awsCLIConfigure, "set", awsCLIRegion, awsConfig.Region}
		if awsConfig.Profile != "" {
			args = append(args, "--profile", awsConfig.Profile)
		}
		cmd := commandContext(ctx, "aws", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set AWS region: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current AWS configuration state.
// Profile resolution order: AWS_PROFILE env → state file → empty.
func (a *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	profile := resolveActiveProfile()

	args := []string{"configure", "get", "region"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	cmd := commandContext(ctx, "aws", args...)
	//nolint:errcheck // best-effort probe; empty string acceptable if unavailable
	regionOutput, _ := cmd.Output()

	return &environment.AWSConfig{
		Profile: profile,
		Region:  strings.TrimSpace(string(regionOutput)),
	}, nil
}

// Rollback rolls back to the previous AWS configuration.
func (a *Switcher) Rollback(ctx context.Context, previousState any) error {
	return a.Switch(ctx, previousState)
}
