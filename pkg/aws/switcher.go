// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"errors"
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
	if !ok || awsConfig == nil {
		return fmt.Errorf("invalid AWS configuration type")
	}

	// Persist active profile. This is the real activation signal for this tool.
	// Do NOT use `aws configure set profile` — that only writes a key into the
	// default profile block and does not change which profile is active.
	if awsConfig.Profile != "" {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("failed to set AWS profile: %w", err)
		}
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
	return a.getCurrentState(ctx, runSnapshotCommand)
}

// snapshotCommand는 호출별 실행 경계이며 새 전역 가변 hook을 만들지 않는다.
type snapshotCommand func(context.Context, string, ...string) ([]byte, error)

func (a *Switcher) getCurrentState(ctx context.Context, run snapshotCommand) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("aws state snapshot: %w", err)
	}
	profile, err := resolveActiveProfileStrict()
	if err != nil {
		return nil, fmt.Errorf("aws profile state: %w", err)
	}
	args := []string{"configure", "get", "region"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	region, err := snapshotValue(ctx, run, "region", args...)
	if err != nil {
		return nil, err
	}
	return &environment.AWSConfig{Profile: profile, Region: region}, nil
}

func runSnapshotCommand(ctx context.Context, _ string, args ...string) ([]byte, error) {
	cmd := commandContext(ctx, "aws", args...)
	return cmd.Output()
}

func snapshotValue(ctx context.Context, run snapshotCommand, query string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("aws %s state query: %w", query, err)
	}
	output, err := run(ctx, "aws", args...)
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("aws %s state query: %w", query, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Rollback rolls back to the previous AWS configuration.
func (a *Switcher) Rollback(ctx context.Context, previousState any) error {
	return a.Switch(ctx, previousState)
}
