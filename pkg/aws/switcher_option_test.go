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
