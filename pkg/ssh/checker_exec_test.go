// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

func withLookPath(t *testing.T, available bool) {
	t.Helper()
	orig := lookPath
	lookPath = func(file string) (string, error) {
		if available {
			return "/usr/bin/" + file, nil
		}
		return "", fmt.Errorf("%s not found", file)
	}
	t.Cleanup(func() { lookPath = orig })
}

type cmdResponse struct {
	stdout string
	code   int
}

func (r cmdResponse) toCmd(ctx context.Context) *exec.Cmd {
	if r.code == 0 {
		return exec.CommandContext(ctx, "printf", "%s", r.stdout)
	}
	return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("exit %d", r.code))
}

func withCommandMap(t *testing.T, responses map[string]cmdResponse) {
	t.Helper()
	orig := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		key := strings.Join(append([]string{name}, args...), " ")
		for prefix, resp := range responses {
			if strings.Contains(key, prefix) {
				return resp.toCmd(ctx)
			}
		}
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { commandContext = orig })
}

func TestChecker_CheckStatus_CLIMissing(t *testing.T) {
	withLookPath(t, false)
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v", st.Status)
	}
	if st.Details["error"] != "SSH not found" {
		t.Errorf("error = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_AgentNotRunning_NoAuthSock(t *testing.T) {
	withLookPath(t, true)
	t.Setenv("SSH_AUTH_SOCK", "")
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v", st.Status)
	}
	if st.Details["error"] != "SSH agent not running" {
		t.Errorf("error = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_AgentExitCode2(t *testing.T) {
	withLookPath(t, true)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {code: 2},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive when agent exit 2", st.Status)
	}
}

func TestChecker_CheckStatus_NoKeysLoaded(t *testing.T) {
	withLookPath(t, true)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")
	// First ssh-add -l for agent check succeeds (exit 0), second for keys returns exit 1.
	// Map matches both; use exit 1 so agent check treats exit!=2 as running, and getLoadedKeys returns empty.
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {code: 1},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Details["error"] != "No SSH keys loaded" {
		t.Errorf("error = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_ActiveWithKeys(t *testing.T) {
	withLookPath(t, true)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {stdout: "256 SHA256:abc user@host (ED25519)\n2048 SHA256:def old@host (RSA)\n"},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusActive {
		t.Errorf("Status = %v, want active", st.Status)
	}
	if !strings.Contains(st.Current.Context, "2 keys") {
		t.Errorf("Context = %q", st.Current.Context)
	}
	if !st.Credentials.Valid {
		t.Error("Credentials.Valid = false")
	}
}

func TestChecker_CheckStatus_RSAOnlyWarning(t *testing.T) {
	withLookPath(t, true)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {stdout: "2048 SHA256:def old@host (RSA)\n"},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Credentials.Warning == "" {
		t.Error("expected RSA-only warning")
	}
	if !strings.Contains(st.Credentials.Warning, "Ed25519") {
		t.Errorf("Warning = %q", st.Credentials.Warning)
	}
}

func TestChecker_CheckHealth_SuccessWithConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {stdout: "256 SHA256:abc (ED25519)\n"},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusActive {
		t.Errorf("Status = %v", health.Status)
	}
	if health.Details["config_file"] == nil {
		t.Error("expected config_file detail")
	}
}

func TestChecker_CheckHealth_Failure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {code: 2},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusError {
		t.Errorf("Status = %v", health.Status)
	}
}

func TestChecker_checkSSHKeys_Empty(t *testing.T) {
	cred := NewChecker().checkSSHKeys(nil)
	if cred.Valid {
		t.Error("Valid should be false for empty keys")
	}
	if cred.Warning != "No SSH keys loaded" {
		t.Errorf("Warning = %q", cred.Warning)
	}
}

func TestChecker_getLoadedKeys_HardError(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"ssh-add -l": {code: 2},
	})
	_, err := NewChecker().getLoadedKeys(context.Background())
	if err == nil {
		t.Fatal("expected error for agent exit 2")
	}
}

func TestSwitcher_Switch_DirectoryPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "ssh-configs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: dir})
	if err == nil {
		t.Fatal("expected error for directory path")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSwitcher_Switch_TildePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(sshDir, "config.work")
	if err := os.WriteFile(src, []byte("Host work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: "~/.ssh/config.work"})
	if err != nil {
		t.Fatalf("Switch() tilde path error = %v", err)
	}
	active := filepath.Join(sshDir, "config")
	target, err := os.Readlink(active)
	if err != nil {
		t.Fatalf("expected symlink at active config: %v", err)
	}
	if filepath.Clean(target) != filepath.Clean(src) {
		t.Errorf("symlink target = %q, want %q", target, src)
	}
}

func TestSwitcher_Switch_CreatesSymlinkWhenAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(sshDir, "config.a")
	if err := os.WriteFile(src, []byte("Host a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: src}); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	active := filepath.Join(sshDir, "config")
	if _, err := os.Readlink(active); err != nil {
		t.Fatalf("expected symlink: %v", err)
	}
}

func TestResolveConfigPath_Empty(t *testing.T) {
	_, err := resolveConfigPath("  ")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestSameFile_Missing(t *testing.T) {
	if sameFile("/no/such/a", "/no/such/b") {
		t.Error("sameFile should be false for missing paths")
	}
}
