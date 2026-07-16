// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"os"
	"path/filepath"
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
	if got := switcher.Name(); got != "ssh" {
		t.Errorf("Name() = %q, want %q", got, "ssh")
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

	if err.Error() != "invalid SSH configuration type" {
		t.Errorf("Switch() error = %q, want %q", err.Error(), "invalid SSH configuration type")
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

// TestSwitcher_Switch_EmptyConfigPath refuses silent success with empty path.
func TestSwitcher_Switch_EmptyConfigPath(t *testing.T) {
	switcher := NewSwitcher()
	ctx := context.Background()

	err := switcher.Switch(ctx, &environment.SSHConfig{Config: ""})
	if err == nil {
		t.Fatal("Switch() with empty config path should return error")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Switch() error = %q, want required-path message", err.Error())
	}
}

// TestSwitcher_Switch_MissingConfigPath fails when the file does not exist.
func TestSwitcher_Switch_MissingConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	switcher := NewSwitcher()
	ctx := context.Background()
	missing := filepath.Join(home, "no-such-ssh-config")

	err := switcher.Switch(ctx, &environment.SSHConfig{Config: missing})
	if err == nil {
		t.Fatal("Switch() with missing config path should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Switch() error = %q, want not-found message", err.Error())
	}
}

// TestSwitcher_Switch_AlreadyActive succeeds when path is the active config file.
func TestSwitcher_Switch_AlreadyActive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(sshDir, "config")
	if err := os.WriteFile(active, []byte("Host *\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	switcher := NewSwitcher()
	ctx := context.Background()

	if err := switcher.Switch(ctx, &environment.SSHConfig{Config: active}); err != nil {
		t.Fatalf("Switch() already-active path error = %v", err)
	}
}

// TestSwitcher_Switch_ManagedSymlink retargets a managed symlink layout.
func TestSwitcher_Switch_ManagedSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}

	work := filepath.Join(sshDir, "config.work")
	prod := filepath.Join(sshDir, "config.prod")
	if err := os.WriteFile(work, []byte("Host work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prod, []byte("Host prod\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	active := filepath.Join(sshDir, "config")
	if err := os.Symlink(work, active); err != nil {
		t.Fatal(err)
	}

	switcher := NewSwitcher()
	ctx := context.Background()

	if err := switcher.Switch(ctx, &environment.SSHConfig{Config: prod}); err != nil {
		t.Fatalf("Switch() managed symlink error = %v", err)
	}

	target, err := os.Readlink(active)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if filepath.Clean(target) != filepath.Clean(prod) {
		t.Errorf("active symlink target = %q, want %q", target, prod)
	}

	state, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}
	sshCfg := state.(*environment.SSHConfig)
	if filepath.Clean(sshCfg.Config) != filepath.Clean(prod) {
		t.Errorf("GetCurrentState().Config = %q, want %q", sshCfg.Config, prod)
	}
}

// TestSwitcher_Switch_RefuseClobberRegularFile fails honestly instead of ignoring config.
func TestSwitcher_Switch_RefuseClobberRegularFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(sshDir, "config")
	if err := os.WriteFile(active, []byte("Host current\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(sshDir, "other")
	if err := os.WriteFile(other, []byte("Host other\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	switcher := NewSwitcher()
	err := switcher.Switch(context.Background(), &environment.SSHConfig{Config: other})
	if err == nil {
		t.Fatal("Switch() should refuse to clobber a regular-file active config")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("Switch() error = %q, want refuse-overwrite message", err.Error())
	}
}

// TestSwitcher_GetCurrentState_NoConfig returns empty path, never hardcoded "default".
func TestSwitcher_GetCurrentState_NoConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	switcher := NewSwitcher()
	state, err := switcher.GetCurrentState(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	sshConfig, ok := state.(*environment.SSHConfig)
	if !ok {
		t.Fatalf("GetCurrentState() returned %T, want *environment.SSHConfig", state)
	}
	if sshConfig.Config == "default" {
		t.Fatal(`GetCurrentState() must not hardcode "default"`)
	}
	if sshConfig.Config != "" {
		t.Errorf("GetCurrentState().Config = %q, want empty when no config", sshConfig.Config)
	}
}

// TestSwitcher_GetCurrentState_ReportsExistingPath returns the real config path.
func TestSwitcher_GetCurrentState_ReportsExistingPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(sshDir, "config")
	if err := os.WriteFile(active, []byte("Host *\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	switcher := NewSwitcher()
	state, err := switcher.GetCurrentState(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	sshConfig := state.(*environment.SSHConfig)
	if sshConfig.Config == "default" {
		t.Fatal(`GetCurrentState() must not hardcode "default"`)
	}
	if filepath.Clean(sshConfig.Config) != filepath.Clean(active) {
		t.Errorf("GetCurrentState().Config = %q, want %q", sshConfig.Config, active)
	}
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

// TestSwitcher_Rollback_ManagedSymlink restores previous symlink target.
func TestSwitcher_Rollback_ManagedSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(sshDir, "config.work")
	prod := filepath.Join(sshDir, "config.prod")
	if err := os.WriteFile(work, []byte("Host work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prod, []byte("Host prod\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(sshDir, "config")
	if err := os.Symlink(work, active); err != nil {
		t.Fatal(err)
	}

	switcher := NewSwitcher()
	ctx := context.Background()

	prev, err := switcher.GetCurrentState(ctx)
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}

	if err := switcher.Switch(ctx, &environment.SSHConfig{Config: prod}); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	if err := switcher.Rollback(ctx, prev); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	target, err := os.Readlink(active)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if filepath.Clean(target) != filepath.Clean(work) {
		t.Errorf("after rollback target = %q, want %q", target, work)
	}
}
