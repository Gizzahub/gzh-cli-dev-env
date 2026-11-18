// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// TestSwitch_RejectsOptionLikeRegionOrProfile proves an option-like region or
// profile fails before the profile state file is written or aws runs.
func TestSwitch_RejectsOptionLikeRegionOrProfile(t *testing.T) {
	tests := map[string]*environment.AWSConfig{
		"region":       {Profile: "dev", Region: "-h"},
		"profile":      {Profile: "--debug", Region: "ap-northeast-2"},
		"profile only": {Profile: "--debug"},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			cfgDir := withTempConfigDir(t)
			captured := withFakeAWS(t)
			t.Setenv("AWS_PROFILE", "")
			err := NewSwitcher().Switch(context.Background(), cfg)
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("Switch() error = %v, want ErrOptionLikeValue", err)
			}
			if len(*captured) != 0 {
				t.Fatalf("aws ran despite rejection: %v", *captured)
			}
			statePath := filepath.Join(cfgDir, configAppDir, activeProfileFileName)
			if _, statErr := os.Stat(statePath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("profile state file touched despite rejection: %v", statErr)
			}
		})
	}
}

// TestGetCurrentState_RejectsOptionLikeActiveProfile proves an option-like
// profile from AWS_PROFILE or the state file fails before any aws command runs.
func TestGetCurrentState_RejectsOptionLikeActiveProfile(t *testing.T) {
	for _, source := range []string{"env", "state file"} {
		t.Run(source, func(t *testing.T) {
			cfgDir := withTempConfigDir(t)
			captured := withFakeAWS(t)
			t.Setenv("AWS_PROFILE", "")
			if source == "env" {
				t.Setenv("AWS_PROFILE", "--debug")
			} else {
				dir := filepath.Join(cfgDir, configAppDir)
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, activeProfileFileName), []byte("--debug\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var ran [][]string
			run := func(_ context.Context, name string, args ...string) ([]byte, error) {
				ran = append(ran, append([]string{name}, args...))
				return []byte("ap-northeast-2\n"), nil
			}
			_, err := NewSwitcher().getCurrentState(context.Background(), run)
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("getCurrentState() error = %v, want ErrOptionLikeValue", err)
			}
			if len(ran) != 0 || len(*captured) != 0 {
				t.Fatalf("aws ran despite rejection: %v %v", ran, *captured)
			}
		})
	}
}
