// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// TestNewSwitcher verifies the constructor creates a valid switcher.
func TestNewSwitcher(t *testing.T) {
	switcher := NewSwitcher()
	if switcher == nil {
		t.Fatal("NewSwitcher() returned nil")
	}
}

// TestSwitcher_Name verifies the service name.
func TestSwitcher_Name(t *testing.T) {
	switcher := NewSwitcher()
	if got := switcher.Name(); got != "aws" {
		t.Errorf("Name() = %q, want %q", got, "aws")
	}
}

// TestSwitcher_ImplementsInterface verifies Switcher implements ServiceSwitcher.
func TestSwitcher_ImplementsInterface(t *testing.T) {
	var _ environment.ServiceSwitcher = (*Switcher)(nil)
}

// TestSwitcher_Switch_InvalidConfigType tests error handling for invalid config type.
func TestSwitcher_Switch_InvalidConfigType(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := switcher.Switch(ctx, "invalid-config")
	if err == nil {
		t.Error("Switch() with invalid config should return error")
	}

	if err.Error() != "invalid AWS configuration type" {
		t.Errorf("Switch() error = %q, want %q", err.Error(), "invalid AWS configuration type")
	}
}

// TestSwitcher_Switch_NilConfig tests error handling for nil config.
func TestSwitcher_Switch_NilConfig(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := switcher.Switch(ctx, nil)
	if err == nil {
		t.Error("Switch() with nil config should return error")
	}
}

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = orig })
	return dir
}

func withFakeAWS(t *testing.T) *[][]string {
	t.Helper()
	var captured [][]string
	orig := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		captured = append(captured, append([]string{name}, args...))
		// Succeed without invoking real aws.
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { commandContext = orig })
	return &captured
}

// TestSwitcher_Switch_PersistsProfileState_andNeverUsesConfigureSetProfile
// proves profile activation is state-file based, not `aws configure set profile`.
func TestSwitcher_Switch_PersistsProfileState_andNeverUsesConfigureSetProfile(t *testing.T) {
	// Given
	cfgDir := withTempConfigDir(t)
	captured := withFakeAWS(t)
	t.Setenv("AWS_PROFILE", "")
	switcher := NewSwitcher()
	ctx := context.Background()
	wantProfile := "prod-account"
	wantRegion := "ap-northeast-2"

	// When
	err := switcher.Switch(ctx, &environment.AWSConfig{
		Profile: wantProfile,
		Region:  wantRegion,
	})

	// Then
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	statePath := filepath.Join(cfgDir, configAppDir, activeProfileFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state file not written: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != wantProfile {
		t.Errorf("state file profile = %q, want %q", got, wantProfile)
	}

	for _, argv := range *captured {
		// Forbidden: aws configure set profile <name>
		if len(argv) >= 4 && argv[0] == "aws" && argv[1] == "configure" && argv[2] == "set" && argv[3] == "profile" {
			t.Fatalf("Switch must not run `aws configure set profile`; got %v", argv)
		}
	}

	// Region must be set with --profile
	foundRegion := false
	for _, argv := range *captured {
		if len(argv) >= 5 &&
			argv[0] == "aws" &&
			argv[1] == "configure" &&
			argv[2] == "set" &&
			argv[3] == "region" &&
			argv[4] == wantRegion &&
			slices.Contains(argv, "--profile") &&
			slices.Contains(argv, wantProfile) {
			foundRegion = true
		}
	}
	if !foundRegion {
		t.Fatalf("expected region set with --profile; captured=%v", *captured)
	}
}

// TestSwitcher_GetCurrentState_PrefersEnvThenStateFile covers profile resolution order.
func TestSwitcher_GetCurrentState_PrefersEnvThenStateFile(t *testing.T) {
	cfgDir := withTempConfigDir(t)
	_ = withFakeAWS(t)
	switcher := NewSwitcher()
	ctx := context.Background()

	t.Run("env wins over state file", func(t *testing.T) {
		// Given
		if err := writeActiveProfile("from-state"); err != nil {
			t.Fatalf("writeActiveProfile: %v", err)
		}
		t.Setenv("AWS_PROFILE", "from-env")

		// When
		state, err := switcher.GetCurrentState(ctx)
		if err != nil {
			t.Fatalf("GetCurrentState() error = %v", err)
		}

		// Then
		cfg := state.(*environment.AWSConfig)
		if cfg.Profile != "from-env" {
			t.Errorf("Profile = %q, want %q", cfg.Profile, "from-env")
		}
	})

	t.Run("state file when env empty", func(t *testing.T) {
		// Given
		t.Setenv("AWS_PROFILE", "")
		if err := writeActiveProfile("state-only"); err != nil {
			t.Fatalf("writeActiveProfile: %v", err)
		}

		// When
		state, err := switcher.GetCurrentState(ctx)
		if err != nil {
			t.Fatalf("GetCurrentState() error = %v", err)
		}

		// Then
		cfg := state.(*environment.AWSConfig)
		if cfg.Profile != "state-only" {
			t.Errorf("Profile = %q, want %q", cfg.Profile, "state-only")
		}
	})

	t.Run("empty when neither set", func(t *testing.T) {
		// Given
		t.Setenv("AWS_PROFILE", "")
		_ = os.Remove(filepath.Join(cfgDir, configAppDir, activeProfileFileName))

		// When
		state, err := switcher.GetCurrentState(ctx)
		if err != nil {
			t.Fatalf("GetCurrentState() error = %v", err)
		}

		// Then
		cfg := state.(*environment.AWSConfig)
		if cfg.Profile != "" {
			t.Errorf("Profile = %q, want empty", cfg.Profile)
		}
	})
}

// TestSwitcher_Switch_RoundTrip_StateFile verifies Switch then GetCurrentState without AWS_PROFILE.
func TestSwitcher_Switch_RoundTrip_StateFile(t *testing.T) {
	// Given
	withTempConfigDir(t)
	_ = withFakeAWS(t)
	t.Setenv("AWS_PROFILE", "")
	switcher := NewSwitcher()
	ctx := context.Background()

	// When
	if err := switcher.Switch(ctx, &environment.AWSConfig{Profile: "dev"}); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	// Then
	cfg := state.(*environment.AWSConfig)
	if cfg.Profile != "dev" {
		t.Errorf("Profile after switch = %q, want %q", cfg.Profile, "dev")
	}
}

// TestSwitcher_GetCurrentState tests GetCurrentState returns valid structure.
func TestSwitcher_GetCurrentState(t *testing.T) {
	withTempConfigDir(t)
	_ = withFakeAWS(t)
	t.Setenv("AWS_PROFILE", "")
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}
	if state == nil {
		t.Fatal("GetCurrentState() returned nil")
	}
	awsConfig, ok := state.(*environment.AWSConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.AWSConfig", state)
	}
	_ = awsConfig.Profile
	_ = awsConfig.Region
}

// TestSwitcher_Rollback_ValidState tests rollback with valid state.
func TestSwitcher_Rollback_ValidState(t *testing.T) {
	withTempConfigDir(t)
	_ = withFakeAWS(t)
	t.Setenv("AWS_PROFILE", "")
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}
	err = switcher.Rollback(ctx, state)
	_ = err
}

// TestSwitcher_Rollback_InvalidState tests rollback with invalid state type.
func TestSwitcher_Rollback_InvalidState(t *testing.T) {
	switcher := NewSwitcher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := switcher.Rollback(ctx, "invalid-state")
	if err == nil {
		t.Error("Rollback() with invalid state should return error")
	}
}

// TestWriteReadActiveProfile_RoundTrip unit-tests state file helpers.
func TestWriteReadActiveProfile_RoundTrip(t *testing.T) {
	withTempConfigDir(t)

	if err := writeActiveProfile("  staging  "); err != nil {
		t.Fatalf("writeActiveProfile: %v", err)
	}
	got, err := readActiveProfile()
	if err != nil {
		t.Fatalf("readActiveProfile: %v", err)
	}
	if got != "staging" {
		t.Errorf("readActiveProfile() = %q, want %q", got, "staging")
	}
}
